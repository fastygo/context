// Package chunk defines Chunker ports and paragraph/markdown chunkers.
package chunk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/parse"
)

// RawChunk is a chunker output before IDs and Merkle assignment.
type RawChunk struct {
	Span         foundation.ByteSpan
	Text         string
	TextChecksum foundation.ChecksumHex
	HeadingPath  []string
	Boundaries   []parse.Boundary
}

// Chunker splits a parsed document into indexed spans.
type Chunker interface {
	Version() string
	Chunk(ctx context.Context, doc parse.Document) ([]RawChunk, error)
}

// Enricher optionally attaches boundary metadata; phase-1 may be a no-op.
type Enricher interface {
	Enrich(ctx context.Context, doc parse.Document, chunks []RawChunk) ([]RawChunk, error)
}

// Paragraph splits on blank-line separated paragraphs.
type Paragraph struct{}

func (Paragraph) Version() string { return "paragraph-v1" }

func (Paragraph) Chunk(ctx context.Context, doc parse.Document) ([]RawChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if doc.Text == "" {
		return nil, nil
	}
	var out []RawChunk
	for _, b := range doc.Boundaries {
		if b.Kind != parse.BoundaryParagraph {
			continue
		}
		rc, err := rawFromSpan(doc.Text, b.Start, b.End, nil, []parse.Boundary{b})
		if err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	if len(out) == 0 {
		rc, err := rawFromSpan(doc.Text, 0, uint64(len(doc.Text)), nil, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	return out, nil
}

// MarkdownSection chunks by markdown heading sections when present.
type MarkdownSection struct {
	Fallback Paragraph
}

func (MarkdownSection) Version() string { return "markdown-section-v1" }

func (c MarkdownSection) Chunk(ctx context.Context, doc parse.Document) ([]RawChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(doc.Sections) == 0 {
		return c.Fallback.Chunk(ctx, doc)
	}
	// Prefer top-level sections (lowest level number among starts), else each section body.
	var out []RawChunk
	seen := map[foundation.ByteSpan]bool{}
	for _, sec := range doc.Sections {
		if sec.Level != 1 && sec.Level != 2 {
			continue
		}
		bodyStart := sec.Start
		// Skip the heading line itself when possible.
		if nl := strings.IndexByte(doc.Text[sec.Start:], '\n'); nl >= 0 {
			bodyStart = sec.Start + uint64(nl) + 1
		}
		if bodyStart >= sec.End {
			continue
		}
		span := foundation.ByteSpan{Start: bodyStart, End: sec.End}
		for span.End > span.Start && (doc.Text[span.End-1] == '\n' || doc.Text[span.End-1] == ' ') {
			span.End--
		}
		if err := span.Validate(); err != nil {
			continue
		}
		if seen[span] {
			continue
		}
		seen[span] = true
		rc, err := rawFromSpan(doc.Text, span.Start, span.End, sec.Ancestry, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	if len(out) == 0 {
		return c.Fallback.Chunk(ctx, doc)
	}
	return out, nil
}

// NoopEnricher returns chunks unchanged.
type NoopEnricher struct{}

func (NoopEnricher) Enrich(_ context.Context, _ parse.Document, chunks []RawChunk) ([]RawChunk, error) {
	return chunks, nil
}

func rawFromSpan(text string, start, end uint64, heading []string, bounds []parse.Boundary) (RawChunk, error) {
	span := foundation.ByteSpan{Start: start, End: end}
	if err := span.Validate(); err != nil {
		return RawChunk{}, apperr.Wrap(apperr.Validation, "chunk span", err)
	}
	if end > uint64(len(text)) {
		return RawChunk{}, apperr.New(apperr.Validation, "chunk span exceeds document")
	}
	piece := text[start:end]
	sum := sha256.Sum256([]byte(piece))
	return RawChunk{
		Span:         span,
		Text:         piece,
		TextChecksum: foundation.ChecksumHex(hex.EncodeToString(sum[:])),
		HeadingPath:  append([]string(nil), heading...),
		Boundaries:   bounds,
	}, nil
}
