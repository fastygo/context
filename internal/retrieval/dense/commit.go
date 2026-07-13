package dense

import (
	"context"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/retrieval"
)

// ChunkDoc is one chunk body + version pins for snapshot-commit dense upsert.
type ChunkDoc struct {
	ProjectID        ids.ProjectID
	SnapshotID       ids.SnapshotID
	ChunkID          ids.ChunkID
	Text             string
	Language         string
	ChunkerVersion   string
	EmbeddingVersion string
	MorphVersion     string
	Span             foundation.ByteSpan
}

// UpsertEmbedded embeds docs and writes VectorPoints for the snapshot namespace.
// Callers must run this before sealing/activating a dense-enabled snapshot
// (ADR-0021). Empty docs is a no-op.
func UpsertEmbedded(
	ctx context.Context,
	store retrieval.VectorStore,
	emb models.Embedder,
	ns indexing.VectorNamespace,
	docs []ChunkDoc,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil {
		return apperr.New(apperr.Validation, "dense: vector store required")
	}
	if emb == nil {
		return apperr.New(apperr.Validation, "dense: embedder required")
	}
	if len(docs) == 0 {
		return nil
	}
	if err := ns.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "vector_namespace", err)
	}
	if ns.EmbeddingVersion == "" {
		return apperr.New(apperr.Validation, "embedding_version required")
	}

	texts := make([]string, len(docs))
	for i, d := range docs {
		if err := d.ProjectID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "project_id", err)
		}
		if err := d.SnapshotID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "snapshot_id", err)
		}
		if err := d.ChunkID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "chunk_id", err)
		}
		if d.ProjectID != ns.ProjectID || d.SnapshotID != ns.SnapshotID {
			return apperr.New(apperr.Validation, "dense: doc project/snapshot must match namespace")
		}
		texts[i] = d.Text
	}

	vecs, modelVer, err := emb.Embed(ctx, texts)
	if err != nil {
		return err
	}
	_ = modelVer
	if len(vecs) != len(docs) {
		return apperr.New(apperr.Internal, "dense: embedder returned unexpected vector count")
	}

	points := make([]retrieval.VectorPoint, 0, len(docs))
	for i, d := range docs {
		embVer := d.EmbeddingVersion
		if embVer == "" {
			embVer = ns.EmbeddingVersion
		}
		if embVer != ns.EmbeddingVersion {
			return apperr.New(apperr.Validation, "dense: chunk embedding_version must match namespace")
		}
		chunker := d.ChunkerVersion
		if chunker == "" {
			chunker = "unknown"
		}
		morph := d.MorphVersion
		if morph == "" {
			morph = "noop-v1"
		}
		points = append(points, retrieval.VectorPoint{
			ChunkID:          d.ChunkID,
			ProjectID:        d.ProjectID,
			SnapshotID:       d.SnapshotID,
			EmbeddingVersion: embVer,
			ChunkerVersion:   chunker,
			MorphVersion:     morph,
			Language:         d.Language,
			Span:             d.Span,
			Vector:           vecs[i],
		})
	}
	return store.Upsert(ctx, ns, points)
}
