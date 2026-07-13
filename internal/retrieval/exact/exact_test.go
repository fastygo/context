package exact_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/index"
)

func fixtureIndex() *index.Memory {
	return index.NewMemory(
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c1", SourceID: "s1",
			Span: foundation.ByteSpan{Start: 0, End: 20}, Text: "alpha beta gamma",
			TextChecksum: "t1", TrustLevel: foundation.TrustProject,
		},
		index.ChunkRecord{
			ProjectID: "p1", SnapshotID: "snap1", ChunkID: "c2", SourceID: "s2",
			Span: foundation.ByteSpan{Start: 0, End: 12}, Text: "other text",
			TextChecksum: "t2", TrustLevel: foundation.TrustProject,
		},
	)
}

func TestExactPhraseMatch(t *testing.T) {
	t.Parallel()
	r := exact.Retriever{Index: fixtureIndex()}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
	}
	cands, err := r.Retrieve(context.Background(), plan, "beta gamma")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].ChunkID != "c1" {
		t.Fatalf("cands=%#v", cands)
	}
	if cands[0].Contributions[0].Explanation == "" {
		t.Fatal("expected explanation")
	}
}

func TestExactTypoNegative(t *testing.T) {
	t.Parallel()
	r := exact.Retriever{Index: fixtureIndex()}
	plan := retrieval.RetrievalPlan{
		ID: "plan1", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "exact"}},
	}
	cands, err := r.Retrieve(context.Background(), plan, "bete gamma")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 0 {
		t.Fatalf("expected no hits, got %#v", cands)
	}
}
