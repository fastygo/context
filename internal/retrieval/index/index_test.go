package index_test

import (
	"testing"
	"time"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
)

func TestTemporalFilterUsesDeterministicOverlap(t *testing.T) {
	t.Parallel()
	base := time.Unix(100, 0).UTC()
	record := index.ChunkRecord{
		TemporalMetadata: &corpus.TemporalMetadata{
			Range: corpus.TemporalRange{
				Start: base,
				End:   base.Add(10 * time.Second),
				Basis: corpus.TimeBasisOccurred,
			},
			IngestedAt: base.Add(time.Minute),
		},
	}

	overlap := retrieval.RetrievalFilters{
		TemporalRange: &corpus.TemporalRange{
			Start: base.Add(9 * time.Second),
			End:   base.Add(11 * time.Second),
			Basis: corpus.TimeBasisOccurred,
		},
	}
	if !index.MatchesFilters(record, overlap) {
		t.Fatal("expected overlapping event-time range to match")
	}

	adjacent := overlap
	adjacent.TemporalRange = &corpus.TemporalRange{
		Start: base.Add(10 * time.Second),
		End:   base.Add(20 * time.Second),
		Basis: corpus.TimeBasisOccurred,
	}
	if index.MatchesFilters(record, adjacent) {
		t.Fatal("adjacent half-open range must not match")
	}

	if index.MatchesFilters(index.ChunkRecord{}, overlap) {
		t.Fatal("record without temporal metadata must not match explicit filter")
	}
}
