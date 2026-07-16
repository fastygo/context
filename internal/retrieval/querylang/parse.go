// Package querylang implements the minimal deterministic operator query layer
// (ADR-0043): quoted phrases, AND/OR/NOT, grouping, ~ morphology markers, and
// a lang: directive. Leaves compile onto existing retrieval paths; there is no
// second ranking model and no hidden rewrite — every interpretation step is
// visible in Explain and trace payloads.
package querylang

import (
	"fmt"
	"strings"
)

// Kind labels AST nodes.
type Kind string

const (
	KindAnd    Kind = "and"
	KindOr     Kind = "or"
	KindNot    Kind = "not"
	KindTerm   Kind = "term"
	KindPhrase Kind = "phrase"
)

// Node is one AST node. Term/Phrase nodes carry Text; Morph marks ~ leaves.
type Node struct {
	Kind     Kind
	Children []*Node
	Text     string
	Morph    bool
}

// Query is a parsed operator query.
type Query struct {
	Root *Node
	// Language is the lang: directive value (base subtag, lowered), if any.
	Language string
}

type token struct {
	kind  string // "(" ")" "or" "and" "not" "phrase" "term" "lang"
	text  string
	morph bool
}

// Parse converts an operator query string into a deterministic AST.
func Parse(input string) (Query, error) {
	toks, lang, err := lex(input)
	if err != nil {
		return Query{}, err
	}
	if len(toks) == 0 {
		return Query{}, fmt.Errorf("querylang: no searchable terms")
	}
	p := parser{toks: toks}
	root, err := p.parseOr()
	if err != nil {
		return Query{}, err
	}
	if p.pos != len(p.toks) {
		return Query{}, fmt.Errorf("querylang: unexpected token %q", p.toks[p.pos].text)
	}
	if err := validateRoot(root); err != nil {
		return Query{}, err
	}
	return Query{Root: root, Language: lang}, nil
}

func lex(input string) ([]token, string, error) {
	var toks []token
	lang := ""
	runes := []rune(input)
	i := 0
	n := len(runes)
	for i < n {
		r := runes[i]
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			i++
		case r == '(':
			toks = append(toks, token{kind: "("})
			i++
		case r == ')':
			toks = append(toks, token{kind: ")"})
			i++
		case r == '|':
			toks = append(toks, token{kind: "or"})
			i++
		case r == '-':
			toks = append(toks, token{kind: "not"})
			i++
		case r == '~' || r == '"':
			morph := false
			if r == '~' {
				morph = true
				i++
				if i >= n {
					return nil, "", fmt.Errorf("querylang: dangling ~")
				}
				r = runes[i]
			}
			if r == '"' {
				text, next, err := readQuoted(runes, i)
				if err != nil {
					return nil, "", err
				}
				toks = append(toks, token{kind: "phrase", text: text, morph: morph})
				i = next
			} else {
				text, next := readWord(runes, i)
				if text == "" {
					return nil, "", fmt.Errorf("querylang: dangling ~")
				}
				toks = append(toks, token{kind: "term", text: text, morph: morph})
				i = next
			}
		default:
			text, next := readWord(runes, i)
			i = next
			switch {
			case strings.EqualFold(text, "OR"):
				toks = append(toks, token{kind: "or"})
			case strings.EqualFold(text, "AND"):
				toks = append(toks, token{kind: "and"})
			case strings.EqualFold(text, "NOT"):
				toks = append(toks, token{kind: "not"})
			case strings.HasPrefix(strings.ToLower(text), "lang:"):
				val := strings.ToLower(strings.TrimSpace(text[len("lang:"):]))
				if val == "" {
					return nil, "", fmt.Errorf("querylang: empty lang: directive")
				}
				if idx := strings.IndexAny(val, "-_"); idx > 0 {
					val = val[:idx]
				}
				lang = val
			default:
				toks = append(toks, token{kind: "term", text: text})
			}
		}
	}
	return toks, lang, nil
}

func readQuoted(runes []rune, start int) (string, int, error) {
	// runes[start] == '"'
	i := start + 1
	var sb strings.Builder
	for i < len(runes) {
		if runes[i] == '"' {
			text := strings.TrimSpace(sb.String())
			if text == "" {
				return "", 0, fmt.Errorf("querylang: empty phrase")
			}
			return text, i + 1, nil
		}
		sb.WriteRune(runes[i])
		i++
	}
	return "", 0, fmt.Errorf("querylang: unterminated phrase")
}

func readWord(runes []rune, start int) (string, int) {
	i := start
	var sb strings.Builder
	for i < len(runes) {
		r := runes[i]
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == '(' || r == ')' || r == '"' || r == '|' || r == '~' {
			break
		}
		sb.WriteRune(r)
		i++
	}
	return sb.String(), i
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() (token, bool) {
	if p.pos >= len(p.toks) {
		return token{}, false
	}
	return p.toks[p.pos], true
}

func (p *parser) parseOr() (*Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	children := []*Node{left}
	for {
		t, ok := p.peek()
		if !ok || t.kind != "or" {
			break
		}
		p.pos++
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		children = append(children, right)
	}
	if len(children) == 1 {
		return children[0], nil
	}
	for _, c := range children {
		if c.Kind == KindNot {
			return nil, fmt.Errorf("querylang: OR operand cannot be a pure negation")
		}
	}
	return &Node{Kind: KindOr, Children: children}, nil
}

func (p *parser) parseAnd() (*Node, error) {
	var children []*Node
	for {
		t, ok := p.peek()
		if !ok || t.kind == "or" || t.kind == ")" {
			break
		}
		if t.kind == "and" {
			p.pos++
			continue
		}
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	if len(children) == 0 {
		return nil, fmt.Errorf("querylang: expected term or phrase")
	}
	if len(children) == 1 {
		return children[0], nil
	}
	return &Node{Kind: KindAnd, Children: children}, nil
}

func (p *parser) parseUnary() (*Node, error) {
	t, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("querylang: unexpected end of query")
	}
	if t.kind == "not" {
		p.pos++
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		if child.Kind == KindNot {
			return nil, fmt.Errorf("querylang: double negation is not supported")
		}
		return &Node{Kind: KindNot, Children: []*Node{child}}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (*Node, error) {
	t, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("querylang: unexpected end of query")
	}
	switch t.kind {
	case "(":
		p.pos++
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		closing, ok := p.peek()
		if !ok || closing.kind != ")" {
			return nil, fmt.Errorf("querylang: missing closing parenthesis")
		}
		p.pos++
		return inner, nil
	case "phrase":
		p.pos++
		return &Node{Kind: KindPhrase, Text: t.text, Morph: t.morph}, nil
	case "term":
		p.pos++
		return &Node{Kind: KindTerm, Text: t.text, Morph: t.morph}, nil
	default:
		return nil, fmt.Errorf("querylang: unexpected token %q", t.kind)
	}
}

func validateRoot(root *Node) error {
	switch root.Kind {
	case KindNot:
		return fmt.Errorf("querylang: query cannot be pure negation")
	case KindAnd:
		hasPositive := false
		for _, c := range root.Children {
			if c.Kind != KindNot {
				hasPositive = true
			}
		}
		if !hasPositive {
			return fmt.Errorf("querylang: query needs at least one positive term")
		}
	}
	return nil
}

// Canonical renders the AST deterministically for explain and traces.
func (n *Node) Canonical() string {
	if n == nil {
		return ""
	}
	switch n.Kind {
	case KindTerm:
		if n.Morph {
			return "~" + n.Text
		}
		return n.Text
	case KindPhrase:
		if n.Morph {
			return `~"` + n.Text + `"`
		}
		return `"` + n.Text + `"`
	case KindNot:
		return "(NOT " + n.Children[0].Canonical() + ")"
	case KindAnd, KindOr:
		op := "AND"
		if n.Kind == KindOr {
			op = "OR"
		}
		parts := make([]string, 0, len(n.Children)+1)
		parts = append(parts, op)
		for _, c := range n.Children {
			parts = append(parts, c.Canonical())
		}
		return "(" + strings.Join(parts, " ") + ")"
	default:
		return ""
	}
}
