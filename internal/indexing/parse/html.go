package parse

import (
	"context"
	"html"
	"strings"
	"unicode"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/indexing/normalize"
)

// HTML strips tags to visible text while preserving Original bytes (S3 / A4).
type HTML struct{}

func (HTML) Version() string { return "html-text-v1" }

func (p HTML) Parse(ctx context.Context, mediaType string, original []byte) (Document, error) {
	if err := ctx.Err(); err != nil {
		return Document{}, err
	}
	extracted := stripHTML(string(original))
	text, err := normalize.ForHashing([]byte(extracted))
	if err != nil {
		return Document{}, apperr.Wrap(apperr.Validation, "html parse", err)
	}
	return Document{
		Text:                 text,
		Original:             append([]byte(nil), original...),
		MediaType:            mediaType,
		ParserVersion:        p.Version(),
		Boundaries:           paragraphBoundaries(text),
		ExtractionConfidence: 0.9,
		LowConfidence:        false,
	}, nil
}

func stripHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	inScript := false
	var tagBuf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inTag {
			if c == '>' {
				tag := strings.ToLower(strings.TrimSpace(tagBuf.String()))
				tagBuf.Reset()
				inTag = false
				name := tag
				if idx := strings.IndexAny(name, " \t/"); idx >= 0 {
					name = name[:idx]
				}
				switch name {
				case "script", "style":
					inScript = true
				case "/script", "/style":
					inScript = false
				case "br", "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6":
					if !inScript {
						b.WriteByte('\n')
					}
				}
				continue
			}
			tagBuf.WriteByte(c)
			continue
		}
		if c == '<' {
			inTag = true
			continue
		}
		if inScript {
			continue
		}
		b.WriteByte(c)
	}
	decoded := html.UnescapeString(b.String())
	return collapseSpace(decoded)
}

func collapseSpace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if r == '\n' {
				b.WriteByte('\n')
				prevSpace = true
				continue
			}
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

var _ Parser = HTML{}
