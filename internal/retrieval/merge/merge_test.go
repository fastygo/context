package merge_test

import (
	"testing"

	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/merge"
)

func TestNormalizeScoresMinMax(t *testing.T) {
	t.Parallel()
	got := merge.NormalizeScores([]float64{1, 3, 5})
	if got[0] != 0 || got[1] != 0.5 || got[2] != 1 {
		t.Fatalf("got=%v", got)
	}
	zeros := merge.NormalizeScores([]float64{2, 2})
	if zeros[0] != 0 || zeros[1] != 0 {
		t.Fatalf("equal raw must normalize to 0: %v", zeros)
	}
}

func TestDedupPreservesContributions(t *testing.T) {
	t.Parallel()
	ref := corpus.SourceRef{
		ProjectID: "p1",
		SourceID:  "s1",
		Span:      foundation.ByteSpan{Start: 0, End: 4},
		Checksum:  "aa",
	}
	a := retrieval.Candidate{
		ChunkID:      "c1",
		SourceRef:    ref,
		TextChecksum: "aa",
		Contributions: []retrieval.ScoreContribution{{
			RetrieverID: "exact", NormalizedScore: 1, Weight: 1,
		}},
	}
	b := retrieval.Candidate{
		ChunkID:      "c1",
		SourceRef:    ref,
		TextChecksum: "aa",
		Contributions: []retrieval.ScoreContribution{{
			RetrieverID: "sparse", NormalizedScore: 1, Weight: 0.75,
		}},
	}
	merged := merge.DedupAndMerge([]retrieval.Candidate{a, b})
	if len(merged) != 1 || len(merged[0].Contributions) != 2 {
		t.Fatalf("merged=%#v", merged)
	}
	if merged[0].MergedScore != 1.75 {
		t.Fatalf("score=%v", merged[0].MergedScore)
	}
}
