package rerank_test

import (
	"context"
	"testing"

	modelfake "github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/rerank"
)

func TestIdentityPreservesOrder(t *testing.T) {
	t.Parallel()
	in := []retrieval.Candidate{
		{ChunkID: "a", MergedScore: 0.5},
		{ChunkID: "b", MergedScore: 0.9},
	}
	out, err := (rerank.Identity{}).Rerank(context.Background(), "q", in)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ChunkID != "a" || out[1].ChunkID != "b" {
		t.Fatalf("identity mutated order: %#v", out)
	}
}

func TestModelAdapterReordersByScores(t *testing.T) {
	t.Parallel()
	in := []retrieval.Candidate{
		{ChunkID: "short", MergedScore: 1, Snippet: &retrieval.Snippet{Text: "x"}},
		{ChunkID: "hit", MergedScore: 0.1, Snippet: &retrieval.Snippet{Text: "query match here"}},
	}
	out, err := (rerank.ModelAdapter{Model: modelfake.Reranker{}}).Rerank(context.Background(), "query", in)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ChunkID != "hit" {
		t.Fatalf("want query hit first, got %#v", out)
	}
}
