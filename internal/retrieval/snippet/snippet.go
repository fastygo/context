// Package snippet builds offset-stable citation windows from chunk text (C4 / ADR-0033).
package snippet

import (
	"fmt"
	"strings"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
)

// DefaultBefore/After are byte windows around the first phrase match.
const (
	DefaultBefore = 40
	DefaultAfter  = 40
)

// Options control window size around the matched phrase.
type Options struct {
	Before int // bytes before match start; negative or zero uses DefaultBefore
	After  int // bytes after match end; negative or zero uses DefaultAfter
}

// FindPhrase returns the first case-sensitive substring match as a half-open
// byte span into text, matching exact retrieval phrase semantics.
func FindPhrase(text, query string) (foundation.ByteSpan, bool) {
	query = strings.TrimSpace(query)
	if query == "" || text == "" {
		return foundation.ByteSpan{}, false
	}
	idx := strings.Index(text, query)
	if idx < 0 {
		return foundation.ByteSpan{}, false
	}
	return foundation.ByteSpan{
		Start: uint64(idx),
		End:   uint64(idx + len(query)),
	}, true
}

// Extract builds a snippet window around match within chunkText.
// match must be a valid half-open span into chunkText.
func Extract(chunkText string, checksum foundation.ChecksumHex, match foundation.ByteSpan, query string, opts Options) (retrieval.Snippet, error) {
	if err := match.Validate(); err != nil {
		return retrieval.Snippet{}, err
	}
	if int(match.End) > len(chunkText) {
		return retrieval.Snippet{}, fmt.Errorf("snippet: match end %d exceeds chunk length %d", match.End, len(chunkText))
	}
	before := opts.Before
	if before <= 0 {
		before = DefaultBefore
	}
	after := opts.After
	if after <= 0 {
		after = DefaultAfter
	}
	start := int(match.Start)
	if start > before {
		start -= before
	} else {
		start = 0
	}
	end := int(match.End) + after
	if end > len(chunkText) {
		end = len(chunkText)
	}
	span := foundation.ByteSpan{Start: uint64(start), End: uint64(end)}
	if err := span.Validate(); err != nil {
		return retrieval.Snippet{}, err
	}
	return retrieval.Snippet{
		Text:          chunkText[start:end],
		ChunkSpan:     span,
		Highlights:    []foundation.ByteSpan{match},
		ChunkChecksum: checksum,
		Query:         query,
	}, nil
}

// FromChunk finds the first phrase match and extracts a snippet.
func FromChunk(chunkText string, checksum foundation.ChecksumHex, query string, opts Options) (retrieval.Snippet, bool) {
	match, ok := FindPhrase(chunkText, query)
	if !ok {
		return retrieval.Snippet{}, false
	}
	sn, err := Extract(chunkText, checksum, match, query, opts)
	if err != nil {
		return retrieval.Snippet{}, false
	}
	return sn, true
}

// Attach annotates candidates with snippets from an in-memory chunk index.
// Candidates without a phrase match keep Snippet nil. Existing snippets are kept.
func Attach(cands []retrieval.Candidate, idx *index.Memory, projectID ids.ProjectID, snapshotID ids.SnapshotID, query string, opts Options) []retrieval.Candidate {
	if idx == nil || strings.TrimSpace(query) == "" || len(cands) == 0 {
		return cands
	}
	out := make([]retrieval.Candidate, len(cands))
	copy(out, cands)
	for i := range out {
		if out[i].Snippet != nil {
			continue
		}
		rec, ok := idx.Get(projectID, snapshotID, out[i].ChunkID)
		if !ok || rec.Tombstoned {
			continue
		}
		checksum := out[i].TextChecksum
		if checksum == "" {
			checksum = rec.TextChecksum
		}
		sn, ok := FromChunk(rec.Text, checksum, query, opts)
		if !ok {
			continue
		}
		cp := sn
		out[i].Snippet = &cp
	}
	return out
}
