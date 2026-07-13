// Package token captures neutral token spans for snippets and citations.
package token

import (
	"unicode"
	"unicode/utf8"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/chunk"
	"github.com/fastygo/context/internal/linguistic"
)

const TokenizerVersion = "whitespace-v1"

// Capture extracts whitespace/punctuation-separated tokens from a raw chunk.
// Surface preserves original code points; Normalized applies the same text as surface for this simple tokenizer.
func Capture(projectID ids.ProjectID, sourceID ids.SourceID, chunkID ids.ChunkID, language linguistic.LanguageCode, rc chunk.RawChunk) []linguistic.TokenOccurrence {
	text := rc.Text
	var out []linguistic.TokenOccurrence
	i := 0
	idx := 0
	for i < len(text) {
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		start := i
		i += size
		for i < len(text) {
			r2, size2 := utf8.DecodeRuneInString(text[i:])
			if unicode.IsSpace(r2) {
				break
			}
			i += size2
		}
		surface := text[start:i]
		absStart := rc.Span.Start + uint64(start)
		absEnd := rc.Span.Start + uint64(i)
		out = append(out, linguistic.TokenOccurrence{
			ID:                ids.TokenID(string(chunkID) + ":" + itoa(idx)),
			ProjectID:         projectID,
			SourceID:          sourceID,
			ChunkID:           chunkID,
			Language:          language,
			Script:            "Latn",
			Surface:           surface,
			Normalized:        surface,
			Span:              foundation.ByteSpan{Start: absStart, End: absEnd},
			TokenizerVersion:  TokenizerVersion,
			NormalizerVersion: "identity-v1",
		})
		idx++
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
