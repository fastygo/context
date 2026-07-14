package snippet_test

import (
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/snippet"
)

func TestOffsetsStableAcrossReindexUnchangedBytes(t *testing.T) {
	t.Parallel()
	text := "prefix runners run in the park suffix"
	checksum := foundation.ChecksumHex("h-stable")
	query := "runners"

	first, ok := snippet.FromChunk(text, checksum, query, snippet.Options{Before: 8, After: 8})
	if !ok {
		t.Fatal("expected snippet")
	}
	// Simulate re-index of identical bytes (new index records, same text+checksum).
	second, ok := snippet.FromChunk(text, checksum, query, snippet.Options{Before: 8, After: 8})
	if !ok {
		t.Fatal("expected second snippet")
	}
	if first.ChunkSpan != second.ChunkSpan {
		t.Fatalf("chunk_span drift: %#v vs %#v", first.ChunkSpan, second.ChunkSpan)
	}
	if len(first.Highlights) != 1 || first.Highlights[0] != second.Highlights[0] {
		t.Fatalf("highlight drift: %#v vs %#v", first.Highlights, second.Highlights)
	}
	if first.Text != second.Text || first.ChunkChecksum != second.ChunkChecksum {
		t.Fatalf("text/checksum drift")
	}
	if first.Highlights[0].Start != 7 || first.Highlights[0].End != 14 {
		t.Fatalf("unexpected match span %#v", first.Highlights[0])
	}
}

func TestAttachUsesIndexText(t *testing.T) {
	t.Parallel()
	idx := index.NewMemory(index.ChunkRecord{
		ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c1", SourceID: "s1",
		Span: foundation.ByteSpan{Start: 0, End: 23},
		Text: "runners run in the park", TextChecksum: "h1",
		TrustLevel: foundation.TrustProject,
	})
	cands := []retrieval.Candidate{{
		ChunkID: "c1", TextChecksum: "h1", TrustLevel: foundation.TrustProject,
	}}
	out := snippet.Attach(cands, idx, "p1", "snap1", "runners", snippet.Options{})
	if out[0].Snippet == nil {
		t.Fatal("expected attached snippet")
	}
	if out[0].Snippet.Highlights[0].Start != 0 || out[0].Snippet.Highlights[0].End != 7 {
		t.Fatalf("highlights=%#v", out[0].Snippet.Highlights)
	}
}
