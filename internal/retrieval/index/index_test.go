package index_test

import (
	"testing"
	"time"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
)

func TestNoCrossProjectIndexLeakage(t *testing.T) {
	t.Parallel()
	mem := index.NewMemory(
		index.ChunkRecord{
			ProjectID: "pa", SnapshotID: "s1", ChunkID: "c_a",
			Text: "secret alpha", TextChecksum: "aa", TrustLevel: foundation.TrustProject,
		},
		index.ChunkRecord{
			ProjectID: "pb", SnapshotID: "s1", ChunkID: "c_b",
			Text: "other beta", TextChecksum: "bb", TrustLevel: foundation.TrustProject,
		},
	)
	if got := mem.List("pb", "s1"); len(got) != 1 || got[0].ChunkID != "c_b" {
		t.Fatalf("list pb: %#v", got)
	}
	if _, ok := mem.Get("pb", "s1", "c_a"); ok {
		t.Fatal("project_b must not get project_a chunk")
	}
	if got := mem.List("pa", "s1"); len(got) != 1 || got[0].ChunkID != "c_a" {
		t.Fatalf("list pa: %#v", got)
	}
}

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
