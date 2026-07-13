package corpus_test

import (
	"testing"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

func TestProjectRejectsZeroValue(t *testing.T) {
	t.Parallel()
	if err := (corpus.Project{}).Validate(); err == nil {
		t.Fatal("expected zero project to fail")
	}
}

func TestChunkRequiresSpanAndHashes(t *testing.T) {
	t.Parallel()
	ch := corpus.Chunk{
		ID:             "c1",
		ProjectID:      "p1",
		SourceID:       "s1",
		ArtifactID:     "a1",
		ChunkerVersion: "para-v1",
		Span:           foundation.ByteSpan{Start: 0, End: 10},
		TextChecksum:   "abc",
		ChunkHash:      "def",
	}
	if err := ch.Validate(); err != nil {
		t.Fatal(err)
	}
	ch.Span.End = 0
	if err := ch.Validate(); err == nil {
		t.Fatal("expected invalid span to fail")
	}
}

func TestSourceRefRequiresIDs(t *testing.T) {
	t.Parallel()
	ref := corpus.SourceRef{
		ProjectID: ids.ProjectID("p1"),
		SourceID:  ids.SourceID("s1"),
		Span:      foundation.ByteSpan{Start: 1, End: 2},
		Checksum:  "aa",
	}
	if err := ref.Validate(); err != nil {
		t.Fatal(err)
	}
}
