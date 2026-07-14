// Package rerank provides intentional post-merge ranking adapters (C11 / ADR-0036).
package rerank

import (
	"context"
	"sort"

	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/retrieval"
)

// Identity keeps merge order. It is the documented intentional no-op path so
// hybrid search always has an explicit Reranker hook (not an accidental omission).
type Identity struct{}

func (Identity) Rerank(ctx context.Context, query string, candidates []retrieval.Candidate) ([]retrieval.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = query
	out := make([]retrieval.Candidate, len(candidates))
	copy(out, candidates)
	return out, nil
}

// WeightedResort re-applies MergedScore order (stable). Useful when a prior
// step mutated scores or when tests assert the rerank stage ran.
type WeightedResort struct{}

func (WeightedResort) Rerank(ctx context.Context, query string, candidates []retrieval.Candidate) ([]retrieval.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = query
	out := make([]retrieval.Candidate, len(candidates))
	copy(out, candidates)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MergedScore != out[j].MergedScore {
			return out[i].MergedScore > out[j].MergedScore
		}
		return out[i].ChunkID < out[j].ChunkID
	})
	return out, nil
}

// ModelAdapter bridges models.Reranker (passage scores) onto candidate order.
// Passages must align 1:1 with candidates; empty passages fall back to Identity.
type ModelAdapter struct {
	Model models.Reranker
}

func (a ModelAdapter) Rerank(ctx context.Context, query string, candidates []retrieval.Candidate) ([]retrieval.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.Model == nil || len(candidates) == 0 {
		return Identity{}.Rerank(ctx, query, candidates)
	}
	passages := make([]string, len(candidates))
	for i, c := range candidates {
		if c.Snippet != nil && c.Snippet.Text != "" {
			passages[i] = c.Snippet.Text
		} else {
			passages[i] = string(c.ChunkID) + ":" + string(c.TextChecksum)
		}
	}
	scores, err := a.Model.Rerank(ctx, query, passages)
	if err != nil {
		return nil, err
	}
	out := make([]retrieval.Candidate, len(candidates))
	copy(out, candidates)
	n := len(scores)
	if n > len(out) {
		n = len(out)
	}
	for i := 0; i < n; i++ {
		out[i].MergedScore = scores[i]
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MergedScore != out[j].MergedScore {
			return out[i].MergedScore > out[j].MergedScore
		}
		return out[i].ChunkID < out[j].ChunkID
	})
	return out, nil
}

var (
	_ retrieval.Reranker = Identity{}
	_ retrieval.Reranker = WeightedResort{}
	_ retrieval.Reranker = ModelAdapter{}
)
