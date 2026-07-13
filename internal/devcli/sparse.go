package devcli

import (
	"context"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/ops/failinject"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/fake"
	"github.com/fastygo/context/internal/retrieval/index"
	"github.com/fastygo/context/internal/retrieval/sparse"
	"github.com/fastygo/context/internal/retrieval/sparse/postgresfts"
)

// SparseHandle is an opened sparse backend for CLI ingest/search.
type SparseHandle struct {
	Client     retrieval.SparseSearchClient
	Closer     func()
	BackendID  string
	Kind       config.StoreKind
	UsesFTS    bool
}

// OpenSparse opens fake (default) or Postgres FTS from env config.
// Callers must invoke Closer when finished.
func OpenSparse(ctx context.Context, idx *index.Memory) (SparseHandle, error) {
	if err := failinject.Check(failinject.Sparse); err != nil {
		return SparseHandle{}, err
	}
	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return SparseHandle{}, err
	}
	switch cfg.Sparse.Kind {
	case config.StoreKindPostgresFTS:
		client, err := postgresfts.Open(ctx, cfg.Sparse.DSN)
		if err != nil {
			return SparseHandle{}, err
		}
		if err := client.EnsureSchema(ctx); err != nil {
			client.Close()
			return SparseHandle{}, err
		}
		return SparseHandle{
			Client:    client,
			Closer:    client.Close,
			BackendID: postgresfts.BackendID,
			Kind:      cfg.Sparse.Kind,
			UsesFTS:   true,
		}, nil
	default:
		// Offline / memory: term-overlap fake over the in-memory chunk index.
		// Ingest may open without an index when FTS is not configured.
		if idx == nil {
			return SparseHandle{
				Closer:    func() {},
				BackendID: "fake",
				Kind:      cfg.Sparse.Kind,
				UsesFTS:   false,
			}, nil
		}
		return SparseHandle{
			Client:    fake.SparseClient{Index: idx},
			Closer:    func() {},
			BackendID: "fake",
			Kind:      cfg.Sparse.Kind,
			UsesFTS:   false,
		}, nil
	}
}

// Retriever builds the sparse Retriever for this handle.
func (h SparseHandle) Retriever(idx *index.Memory) retrieval.Retriever {
	explain := "fake sparse term overlap"
	if h.UsesFTS {
		explain = "postgres fts ts_rank_cd"
	}
	return sparse.Retriever{Client: h.Client, Index: idx, Explanation: explain}
}

// UpsertChunks writes chunk bodies into FTS when the handle uses Postgres FTS.
func (h SparseHandle) UpsertChunks(ctx context.Context, projectID ids.ProjectID, chunks []IndexedChunk) error {
	if !h.UsesFTS {
		return nil
	}
	client, ok := h.Client.(*postgresfts.Client)
	if !ok {
		return apperr.New(apperr.Internal, "sparse: expected postgresfts client")
	}
	docs := make([]postgresfts.Document, 0, len(chunks))
	for _, ch := range chunks {
		docs = append(docs, postgresfts.Document{
			ProjectID:  projectID,
			SnapshotID: ch.SnapshotID,
			ChunkID:    ch.ChunkID,
			Body:       ch.Text,
		})
	}
	return client.Upsert(ctx, docs)
}

// EnsureFTSFromIndex upserts all index records for project/snapshot (search-time backfill).
func EnsureFTSFromIndex(
	ctx context.Context,
	client *postgresfts.Client,
	idx *index.Memory,
	projectID ids.ProjectID,
	snapshotID ids.SnapshotID,
) error {
	recs := idx.List(projectID, snapshotID)
	if len(recs) == 0 {
		return nil
	}
	docs := make([]postgresfts.Document, 0, len(recs))
	for _, rec := range recs {
		docs = append(docs, postgresfts.Document{
			ProjectID:  rec.ProjectID,
			SnapshotID: rec.SnapshotID,
			ChunkID:    rec.ChunkID,
			Language:   rec.Language,
			Body:       rec.Text,
		})
	}
	return client.Upsert(ctx, docs)
}
