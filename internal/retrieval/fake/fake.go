// Package fake provides deterministic retrieval doubles for unit tests.
package fake

import (
	"context"
	"math"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/merge"
)

const (
	SparseID = "sparse"
	DenseID  = "dense"
)

// SparseClient is an in-memory keyword overlap sparse search double.
type SparseClient struct {
	Index *index.Memory
}

func (c SparseClient) Search(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID, query string, limit int) ([]retrieval.SparseHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := projectID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := snapshotID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "snapshot_id", err)
	}
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 || c.Index == nil {
		return nil, nil
	}
	var hits []retrieval.SparseHit
	for _, rec := range c.Index.List(projectID, snapshotID) {
		text := strings.ToLower(rec.Text)
		var score float64
		for _, term := range terms {
			if strings.Contains(text, term) {
				score++
			}
		}
		if score == 0 {
			continue
		}
		hits = append(hits, retrieval.SparseHit{ChunkID: rec.ChunkID, Score: score})
	}
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// SparseRetriever wraps SparseClient into the Retriever port.
type SparseRetriever struct {
	Client SparseClient
	Index  *index.Memory
}

func (r SparseRetriever) ID() string { return SparseID }

func (r SparseRetriever) Retrieve(ctx context.Context, plan retrieval.RetrievalPlan, query string) ([]retrieval.Candidate, error) {
	hits, err := r.Client.Search(ctx, plan.ProjectID, plan.SnapshotID, query, plan.TopNRawPool)
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
	var out []retrieval.Candidate
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
				RetrieverID:     SparseID,
				RawScore:        s.hit.Score,
				NormalizedScore: norms[i],
				Weight:          merge.DefaultWeight(SparseID),
				Reasons:         []foundation.ScoreReason{foundation.ReasonSparseTerm},
				Explanation:     "fake sparse term overlap",
				SnapshotID:      plan.SnapshotID,
				ProjectID:       plan.ProjectID,
			}},
		})
	}
	return out, nil
}

// VectorStore is an in-memory dense VectorStore double.
type VectorStore struct {
	points map[string]retrieval.VectorPoint // key: project|snapshot|chunk
}

func NewVectorStore() *VectorStore {
	return &VectorStore{points: make(map[string]retrieval.VectorPoint)}
}

func vectorKey(projectID ids.ProjectID, snapshotID ids.SnapshotID, chunkID ids.ChunkID) string {
	return string(projectID) + "\x00" + string(snapshotID) + "\x00" + string(chunkID)
}

func (s *VectorStore) Upsert(ctx context.Context, ns indexing.VectorNamespace, points []retrieval.VectorPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ns.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "vector_namespace", err)
	}
	for _, p := range points {
		if p.ProjectID != ns.ProjectID || p.SnapshotID != ns.SnapshotID {
			return apperr.New(apperr.Validation, "vector point project/snapshot must match namespace")
		}
		if p.EmbeddingVersion == "" {
			return apperr.New(apperr.Validation, "embedding_version required")
		}
		s.points[vectorKey(p.ProjectID, p.SnapshotID, p.ChunkID)] = p
	}
	return nil
}

func (s *VectorStore) Search(ctx context.Context, ns indexing.VectorNamespace, vector []float32, limit int) ([]retrieval.VectorHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ns.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "vector_namespace", err)
	}
	if ns.EmbeddingVersion == "" {
		return nil, apperr.New(apperr.Validation, "embedding_version required")
	}
	var hits []retrieval.VectorHit
	prefix := string(ns.ProjectID) + "\x00" + string(ns.SnapshotID) + "\x00"
	for k, p := range s.points {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if p.EmbeddingVersion != ns.EmbeddingVersion {
			continue
		}
		hits = append(hits, retrieval.VectorHit{
			ChunkID:          p.ChunkID,
			Score:            cosine(vector, p.Vector),
			EmbeddingVersion: p.EmbeddingVersion,
			ChunkerVersion:   p.ChunkerVersion,
			MorphVersion:     p.MorphVersion,
			ContextRef:       p.ContextRef,
			SnapshotID:       p.SnapshotID,
		})
	}
	if limit > 0 && len(hits) > limit {
		// naive top-N by score
		for i := 0; i < len(hits); i++ {
			for j := i + 1; j < len(hits); j++ {
				if hits[j].Score > hits[i].Score {
					hits[i], hits[j] = hits[j], hits[i]
				}
			}
		}
		hits = hits[:limit]
	}
	return hits, nil
}

func cosine(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// HashEmbed is a deterministic fake embedding from text.
func HashEmbed(text string, dim int) []float32 {
	if dim <= 0 {
		dim = 8
	}
	out := make([]float32, dim)
	for i, r := range text {
		out[i%dim] += float32(int(r)%31) / 31
	}
	return out
}

var (
	_ retrieval.SparseSearchClient = SparseClient{}
	_ retrieval.VectorStore        = (*VectorStore)(nil)
	_ retrieval.Retriever          = SparseRetriever{}
)
