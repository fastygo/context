// Package parse defines Parser ports and plaintext/markdown parsers.
package parse

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/indexing/normalize"
)

// BoundaryKind labels structural spans inside a document.
type BoundaryKind string

const (
	BoundaryParagraph       BoundaryKind = "paragraph"
	BoundarySentence        BoundaryKind = "sentence"
	BoundaryHeading         BoundaryKind = "heading"
	BoundaryCitation        BoundaryKind = "citation"
	BoundaryDictionaryEntry BoundaryKind = "dictionary_entry"
	BoundarySense           BoundaryKind = "sense"
	BoundaryAttestation     BoundaryKind = "attestation"
)

// Boundary is a labeled span inside normalized document text.
type Boundary struct {
	Kind  BoundaryKind
	Start uint64
	End   uint64
	Label string
}

// Section captures markdown heading ancestry for section-aware chunking.
type Section struct {
	Level    int
	Title    string
	Start    uint64
	End      uint64
	Ancestry []string
}

// Document is parser output used by chunkers.
type Document struct {
	Text          string // normalized for hashing/chunking
	Original      []byte
	MediaType     string
	ParserVersion string
	Sections      []Section
	Boundaries    []Boundary
	// ExtractionConfidence is 1.0 for lossless text parsers and <1 for lossy
	// extractors (HTML/PDF). Zero means unset and is treated as 1.0 for legacy.
	ExtractionConfidence float64
	// LowConfidence is true when extraction may have dropped or reordered text.
	LowConfidence bool
}

// Parser converts original bytes into a normalized document.
type Parser interface {
	Version() string
	Parse(ctx context.Context, mediaType string, original []byte) (Document, error)
}

// PlainText parses UTF-8 text documents.
type PlainText struct{}

func (PlainText) Version() string { return "plaintext-v1" }

func (p PlainText) Parse(ctx context.Context, mediaType string, original []byte) (Document, error) {
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	text, err := normalize.ForHashing(original)
	if err != nil {
		return Document{}, apperr.Wrap(apperr.Validation, "plaintext parse", err)
	}
	doc := Document{
		Text:                 text,
		Original:             append([]byte(nil), original...),
		MediaType:            mediaType,
		ParserVersion:        p.Version(),
		Boundaries:           paragraphBoundaries(text),
		ExtractionConfidence: 1.0,
	}
	return doc, nil
}

// Markdown parses Markdown with basic ATX heading ancestry.
type Markdown struct{}

func (Markdown) Version() string { return "markdown-v1" }

func (p Markdown) Parse(ctx context.Context, mediaType string, original []byte) (Document, error) {
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	text, err := normalize.ForHashing(original)
	if err != nil {
		return Document{}, apperr.Wrap(apperr.Validation, "markdown parse", err)
	}
	sections := markdownSections(text)
	doc := Document{
		Text:                 text,
		Original:             append([]byte(nil), original...),
		MediaType:            mediaType,
		ParserVersion:        p.Version(),
		Sections:             sections,
		Boundaries:           append(paragraphBoundaries(text), headingBoundaries(sections)...),
		ExtractionConfidence: 1.0,
	}
	return doc, nil
}

// Registry selects a parser by media type.
type Registry struct {
	Plain    PlainText
	Markdown Markdown
	HTML     HTML
	PDF      PDF
}

func (r Registry) For(mediaType string) Parser {
	switch mediaType {
	case "text/markdown":
		return r.Markdown
	case "text/html", "application/xhtml+xml":
		return r.HTML
	case "application/pdf":
		return r.PDF
	default:
		return r.Plain
	}
}

func paragraphBoundaries(text string) []Boundary {
	var out []Boundary
	start := 0
	for i := 0; i <= len(text); {
		if i == len(text) || (i+1 < len(text) && text[i] == '\n' && text[i+1] == '\n') {
			end := i
			for start < end && (text[start] == '\n' || text[start] == ' ' || text[start] == '\t') {
				start++
			}
			for end > start && (text[end-1] == '\n' || text[end-1] == ' ' || text[end-1] == '\t') {
				end--
			}
			if end > start {
				out = append(out, Boundary{Kind: BoundaryParagraph, Start: uint64(start), End: uint64(end)})
			}
			if i == len(text) {
				break
			}
			i += 2
			start = i
			continue
		}
		i++
	}
	return out
}

func markdownSections(text string) []Section {
	lines := splitLinesKeepOffsets(text)
	var stack []Section
	var closed []Section
	type open struct {
		sec   Section
		index int
	}
	var opens []open

	for _, line := range lines {
		level, title, ok := parseATXHeading(line.Text)
		if !ok {
			continue
		}
		for len(opens) > 0 && opens[len(opens)-1].sec.Level >= level {
			last := opens[len(opens)-1]
			opens = opens[:len(opens)-1]
			last.sec.End = uint64(line.Start)
			closed = append(closed, last.sec)
		}
		ancestry := make([]string, 0, len(opens)+1)
		for _, o := range opens {
			ancestry = append(ancestry, o.sec.Title)
		}
		ancestry = append(ancestry, title)
		sec := Section{
			Level:    level,
			Title:    title,
			Start:    uint64(line.Start),
			Ancestry: ancestry,
		}
		opens = append(opens, open{sec: sec})
		_ = stack
	}
	for i := len(opens) - 1; i >= 0; i-- {
		sec := opens[i].sec
		sec.End = uint64(len(text))
		closed = append(closed, sec)
	}
	return closed
}

func headingBoundaries(sections []Section) []Boundary {
	out := make([]Boundary, 0, len(sections))
	for _, s := range sections {
		out = append(out, Boundary{
			Kind:  BoundaryHeading,
			Start: s.Start,
			End:   s.End,
			Label: strings.Join(s.Ancestry, " > "),
		})
	}
	return out
}

type lineSpan struct {
	Text  string
	Start int
}

func splitLinesKeepOffsets(text string) []lineSpan {
	var out []lineSpan
	start := 0
	for i := 0; i <= len(text); i++ {
		if i == len(text) || text[i] == '\n' {
			out = append(out, lineSpan{Text: text[start:i], Start: start})
			start = i + 1
		}
	}
	return out
}

func parseATXHeading(line string) (level int, title string, ok bool) {
	trimmed := strings.TrimRight(line, " \t\r")
	if !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i == 0 || i > 6 {
		return 0, "", false
	}
	if i < len(trimmed) && trimmed[i] != ' ' && trimmed[i] != '\t' {
		return 0, "", false
	}
	title = strings.TrimSpace(trimmed[i:])
	if title == "" {
		return 0, "", false
	}
	return i, title, true
}

// RuneSpan converts a byte span into rune offsets for UI highlighting.
func RuneSpan(text string, start, end uint64) (runeStart, runeEnd uint64) {
	var rstart, rend uint64
	var i uint64
	for i < uint64(len(text)) {
		if i == start {
			rstart = rend
		}
		_, size := utf8.DecodeRuneInString(text[i:])
		i += uint64(size)
		rend++
		if i == end {
			return rstart, rend
		}
	}
	return rstart, rend
}
