package hybrid_test

import (
	"context"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/exact"
	"github.com/fastygo/context/internal/retrieval/hybrid"
)

func TestDenseStrategyUnavailableWhenNil(t *testing.T) {
	t.Parallel()
	eng := hybrid.Engine{
		Exact: exact.Retriever{Index: corpus()},
	}
	plan := retrieval.RetrievalPlan{
		ID: "plan", ProjectID: "p1", SnapshotID: "snap1",
		Strategies: []retrieval.RetrieverStrategy{{RetrieverID: "dense"}},
	}
	_, err := eng.Search(context.Background(), plan, "q", "runners")
	if err == nil || !apperr.Is(err, apperr.Unavailable) {
		t.Fatalf("want unavailable, got %v", err)
	}
}
