// Package sparse provides the sparse Retriever over SparseSearchClient.
package sparse

import (
	"context"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/merge"
)

const RetrieverID = "sparse"

// Retriever maps SparseSearchClient hits through the chunk index so
// temporal/language/lexicon filters stay client-side when the backend cannot
// enforce them.
type Retriever struct {
	Client      retrieval.SparseSearchClient
	Index       *index.Memory
	Explanation string
}

func (r Retriever) ID() string { return RetrieverID }

func (r Retriever) Retrieve(ctx context.Context, plan retrieval.RetrievalPlan, query string) ([]retrieval.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.Client == nil {
		return nil, apperr.New(apperr.Validation, "sparse: client required")
	}
	if r.Index == nil {
		return nil, apperr.New(apperr.Validation, "sparse: chunk index required")
	}
	limit := plan.TopNRawPool
	if limit <= 0 {
		limit = 20
	}
	hits, err := r.Client.Search(ctx, plan.ProjectID, plan.SnapshotID, query, limit)
	if err != nil {
		return nil, err
	}
	type scored struct {
		hit retrieval.SparseHit
		rec index.ChunkRecord
	}
	kept := make([]scored, 0, len(hits))
	for _, h := range hits {
		rec, ok := r.Index.Get(plan.ProjectID, plan.SnapshotID, h.ChunkID)
		if !ok || !index.MatchesFilters(rec, plan.Filters) {
			continue
		}
		kept = append(kept, scored{hit: h, rec: rec})
	}
	raw := make([]float64, len(kept))
	for i, s := range kept {
		raw[i] = s.hit.Score
	}
	norms := merge.NormalizeScores(raw)
	explain := r.Explanation
	if explain == "" {
		explain = "sparse term search"
	}
	out := make([]retrieval.Candidate, 0, len(kept))
	for i, s := range kept {
		out = append(out, retrieval.Candidate{
			ChunkID: s.rec.ChunkID,
			SourceRef: corpus.SourceRef{
				ProjectID: s.rec.ProjectID,
				SourceID:  s.rec.SourceID,
				ChunkID:   s.rec.ChunkID,
				Span:      s.rec.Span,
				Checksum:  s.rec.TextChecksum,
			},
			TrustLevel:   s.rec.TrustLevel,
			TextChecksum: s.rec.TextChecksum,
			Contributions: []retrieval.ScoreContribution{{
				RetrieverID:     RetrieverID,
				RawScore:        s.hit.Score,
				NormalizedScore: norms[i],
				Weight:          merge.DefaultWeight(RetrieverID),
				Reasons:         []foundation.ScoreReason{foundation.ReasonSparseTerm},
				Explanation:     explain,
				SnapshotID:      plan.SnapshotID,
				ProjectID:       plan.ProjectID,
				AnalyzerVersion: s.rec.AnalyzerVersion,
			}},
		})
	}
	return out, nil
}

var _ retrieval.Retriever = Retriever{}
