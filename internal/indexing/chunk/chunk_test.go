package chunk_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/indexing/chunk"
	"github.com/fastygo/context/internal/indexing/parse"
)

func TestParagraphChunkStableSpans(t *testing.T) {
	t.Parallel()
	doc, err := (parse.PlainText{}).Parse(context.Background(), "text/plain", []byte("alpha\n\nbeta"))
	if err != nil {
		t.Fatal(err)
	}
	chunks, err := (chunk.Paragraph{}).Chunk(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks=%d", len(chunks))
	}
	if chunks[0].Text != "alpha" || chunks[1].Text != "beta" {
		t.Fatalf("texts=%q %q", chunks[0].Text, chunks[1].Text)
	}
	if err := chunks[0].Span.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMarkdownSectionChunker(t *testing.T) {
	t.Parallel()
	doc, err := (parse.Markdown{}).Parse(context.Background(), "text/markdown", []byte("# Title\n\nBody one.\n\n## Sub\n\nBody two.\n"))
	if err != nil {
		t.Fatal(err)
	}
	chunks, err := (chunk.MarkdownSection{}).Chunk(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected section chunks")
	}
}
