package corpus_test

import (
	"testing"
	"time"

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

func TestTemporalRangeValidationAndHalfOpenOverlap(t *testing.T) {
	t.Parallel()
	base := time.Unix(100, 0).UTC()
	first := corpus.TemporalRange{
		Start: base,
		End:   base.Add(10 * time.Second),
		Basis: corpus.TimeBasisOccurred,
	}
	overlapping := corpus.TemporalRange{
		Start: base.Add(5 * time.Second),
		End:   base.Add(15 * time.Second),
		Basis: corpus.TimeBasisOccurred,
	}
	adjacent := corpus.TemporalRange{
		Start: base.Add(10 * time.Second),
		End:   base.Add(20 * time.Second),
		Basis: corpus.TimeBasisOccurred,
	}
	if err := first.Validate(); err != nil {
		t.Fatal(err)
	}
	if !first.Overlaps(overlapping) {
		t.Fatal("expected ranges to overlap")
	}
	if first.Overlaps(adjacent) {
		t.Fatal("half-open adjacent ranges must not overlap")
	}
	overlapping.Basis = corpus.TimeBasisObserved
	if first.Overlaps(overlapping) {
		t.Fatal("different time bases must not overlap")
	}
}

func TestTemporalMetadataRejectsMissingIngestedAt(t *testing.T) {
	t.Parallel()
	base := time.Unix(100, 0).UTC()
	metadata := corpus.TemporalMetadata{
		Range: corpus.TemporalRange{
			Start: base,
			End:   base.Add(time.Second),
			Basis: corpus.TimeBasisOccurred,
		},
	}
	if err := metadata.Validate(); err == nil {
		t.Fatal("expected zero ingested_at to fail")
	}
}
