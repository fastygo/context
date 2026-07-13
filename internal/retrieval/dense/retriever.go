// Package dense provides the dense Retriever path over VectorStore + Embedder.
package dense

import (
	"context"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/merge"
)

const RetrieverID = "dense"

// Retriever runs dense similarity search and maps hits through the chunk index
// so temporal/language/lexicon filters stay explainable when the vector backend
// cannot enforce them server-side.
type Retriever struct {
	Store     retrieval.VectorStore
	Embedder  models.Embedder
	Index     *index.Memory
	Namespace indexing.VectorNamespace
}

func (r Retriever) ID() string { return RetrieverID }

func (r Retriever) Retrieve(ctx context.Context, plan retrieval.RetrievalPlan, query string) ([]retrieval.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.Store == nil {
		return nil, apperr.New(apperr.Validation, "dense: vector store required")
	}
	if r.Embedder == nil {
		return nil, apperr.New(apperr.Validation, "dense: embedder required")
	}
	if r.Index == nil {
		return nil, apperr.New(apperr.Validation, "dense: chunk index required")
	}
	ns := r.Namespace
	if ns.ProjectID == "" {
		ns.ProjectID = plan.ProjectID
	}
	if ns.SnapshotID == "" {
		ns.SnapshotID = plan.SnapshotID
	}
	if ns.Name == "" {
		ns.Name = "context_dense_v1"
	}
	if err := ns.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "vector_namespace", err)
	}
	if ns.EmbeddingVersion == "" {
		return nil, apperr.New(apperr.Validation, "embedding_version required")
	}
	if err := plan.ProjectID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := plan.SnapshotID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "snapshot_id", err)
	}
	if ns.ProjectID != plan.ProjectID || ns.SnapshotID != plan.SnapshotID {
		return nil, apperr.New(apperr.Validation, "dense: namespace project/snapshot must match plan")
	}

	vecs, modelVersion, err := r.Embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 {
		return nil, apperr.New(apperr.Internal, "dense: embedder returned unexpected vector count")
	}
	embedVer := ns.EmbeddingVersion
	if modelVersion != "" && modelVersion != embedVer {
		// Prefer namespace pin; record model version on the contribution below.
		_ = modelVersion
	}

	limit := plan.TopNRawPool
	if limit <= 0 {
		limit = 20
	}
	hits, err := r.Store.Search(ctx, ns, vecs[0], limit)
	if err != nil {
		return nil, err
	}

	type scored struct {
		hit retrieval.VectorHit
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
	out := make([]retrieval.Candidate, 0, len(kept))
	for i, s := range kept {
		analyzerVer := s.rec.AnalyzerVersion
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
				Reasons:         []foundation.ScoreReason{foundation.ReasonDenseSimilarity},
				Explanation:     "dense vector similarity",
				SnapshotID:      plan.SnapshotID,
				ProjectID:       plan.ProjectID,
				AnalyzerVersion: analyzerVer,
				EmbedVersion:    firstNonEmpty(s.hit.EmbeddingVersion, embedVer, modelVersion),
			}},
		})
	}
	return out, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

var _ retrieval.Retriever = Retriever{}
