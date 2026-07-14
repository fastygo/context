package parse

import (
	"bytes"
	"context"
	"regexp"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/indexing/normalize"
)

// PDF extracts literal strings from PDF content streams without a full PDF
// library (S3 / A5). ExtractionConfidence is intentionally low.
type PDF struct{}

func (PDF) Version() string { return "pdf-strings-v1" }

var (
	pdfHeader   = []byte("%PDF-")
	tjLiteral   = regexp.MustCompile(`\((?:\\.|[^\\)])*\)\s*Tj`)
	stringLit   = regexp.MustCompile(`\((?:\\.|[^\\)])*\)`)
)

func (p PDF) Parse(ctx context.Context, mediaType string, original []byte) (Document, error) {
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	if !bytes.HasPrefix(bytes.TrimSpace(original), pdfHeader) {
		return Document{}, apperr.New(apperr.Validation, "pdf: missing %PDF- header")
	}
	extracted := extractPDFStrings(original)
	text, err := normalize.ForHashing([]byte(extracted))
	if err != nil {
		return Document{}, apperr.Wrap(apperr.Validation, "pdf parse", err)
	}
	conf := 0.4
	if text == "" {
		conf = 0.1
	}
	return Document{
		Text:                 text,
		Original:             append([]byte(nil), original...),
		MediaType:            mediaType,
		ParserVersion:        p.Version(),
		Boundaries:           paragraphBoundaries(text),
		ExtractionConfidence: conf,
		LowConfidence:        true,
	}, nil
}

func extractPDFStrings(raw []byte) string {
	var parts []string
	for _, m := range tjLiteral.FindAll(raw, -1) {
		for _, lit := range stringLit.FindAll(m, 1) {
			parts = append(parts, unescapePDFString(string(lit)))
		}
	}
	if len(parts) == 0 {
		// Fallback: any parentheses strings outside binary noise.
		for _, lit := range stringLit.FindAll(raw, -1) {
			s := unescapePDFString(string(lit))
			if looksLikeText(s) {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, " ")
}

func unescapePDFString(lit string) string {
	if len(lit) < 2 || lit[0] != '(' || lit[len(lit)-1] != ')' {
		return ""
	}
	inner := lit[1 : len(lit)-1]
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			i++
			switch inner[i] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '(', ')', '\\':
				b.WriteByte(inner[i])
			default:
				b.WriteByte(inner[i])
			}
			continue
		}
		b.WriteByte(inner[i])
	}
	return b.String()
}

func looksLikeText(s string) bool {
	if len(s) < 3 {
		return false
	}
	letters := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letters++
		}
	}
	return letters >= 3
}

var _ Parser = PDF{}
