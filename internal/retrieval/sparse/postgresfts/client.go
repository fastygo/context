// Package postgresfts is the first live SparseSearchClient (ADR-0017).
package postgresfts

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
)

const (
	BackendID = "postgres_fts"
	// TextConfig is the PostgreSQL text search configuration. "simple" avoids
	// English stemming lock-in; morphology-aware sparse needs context-lang-* /
	// context-sparse later (documented limits).
	TextConfig = "simple"
)

// Document is one chunk body row for FTS upsert.
type Document struct {
	ProjectID  ids.ProjectID
	SnapshotID ids.SnapshotID
	ChunkID    ids.ChunkID
	Language   string
	Body       string
}

// Client stores chunk text and searches with Postgres full-text.
type Client struct {
	pool *pgxpool.Pool
}

// Open connects to PostgreSQL. Call EnsureSchema before Upsert/Search.
func Open(ctx context.Context, dsn string) (*Client, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, apperr.New(apperr.Validation, "postgresfts: dsn required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, apperr.Wrap(apperr.Unavailable, "postgresfts connect", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, apperr.Wrap(apperr.Unavailable, "postgresfts ping", err)
	}
	return &Client{pool: pool}, nil
}

// Close releases the pool.
func (c *Client) Close() {
	if c != nil && c.pool != nil {
		c.pool.Close()
	}
}

// Capabilities reports server-side filter support.
func (c *Client) Capabilities() retrieval.BackendCapabilities {
	return retrieval.BackendCapabilities{
		BackendID:                BackendID,
		Kind:                     string(config.StoreKindPostgresFTS),
		SupportsProjectFilter:    true,
		SupportsSnapshotFilter:   true,
		SupportsTemporalFilter:   false,
		SupportsMetadataFilter:   false, // language/lexicon filters are client-side
		SupportsPayloadNamespace: true,
		Metrics:                  []string{"ts_rank_cd"},
		NamespaceModel:           "shared_table_project_snapshot_filter",
		ManagedService:           false,
	}
}

// EnsureSchema creates the FTS table and GIN index.
func (c *Client) EnsureSchema(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := c.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS context_sparse_fts (
  project_id text NOT NULL,
  snapshot_id text NOT NULL,
  chunk_id text NOT NULL,
  language text NOT NULL DEFAULT '',
  body text NOT NULL,
  tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', body)) STORED,
  PRIMARY KEY (project_id, snapshot_id, chunk_id)
)`)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "postgresfts create table", err)
	}
	_, err = c.pool.Exec(ctx, `
CREATE INDEX IF NOT EXISTS context_sparse_fts_tsv_idx
  ON context_sparse_fts USING GIN (tsv)`)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "postgresfts gin index", err)
	}
	_, err = c.pool.Exec(ctx, `
CREATE INDEX IF NOT EXISTS context_sparse_fts_snap_idx
  ON context_sparse_fts (project_id, snapshot_id)`)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "postgresfts snap index", err)
	}
	return nil
}

// Upsert writes or replaces chunk bodies for sparse search.
func (c *Client) Upsert(ctx context.Context, docs []Document) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(docs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, d := range docs {
		if err := d.ProjectID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "project_id", err)
		}
		if err := d.SnapshotID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "snapshot_id", err)
		}
		if err := d.ChunkID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "chunk_id", err)
		}
		batch.Queue(`
INSERT INTO context_sparse_fts (project_id, snapshot_id, chunk_id, language, body)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (project_id, snapshot_id, chunk_id) DO UPDATE SET
  language = EXCLUDED.language,
  body = EXCLUDED.body
`, string(d.ProjectID), string(d.SnapshotID), string(d.ChunkID), d.Language, d.Body)
	}
	br := c.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return apperr.Wrap(apperr.Internal, "postgresfts upsert", err)
		}
	}
	return nil
}

// Search requires project_id and snapshot_id; ranks with ts_rank_cd.
func (c *Client) Search(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID, query string, limit int) ([]retrieval.SparseHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := projectID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := snapshotID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "snapshot_id", err)
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := c.pool.Query(ctx, `
SELECT chunk_id, ts_rank_cd(tsv, q) AS score
FROM context_sparse_fts,
     plainto_tsquery('simple', $3) AS q
WHERE project_id = $1
  AND snapshot_id = $2
  AND tsv @@ q
ORDER BY score DESC, chunk_id ASC
LIMIT $4
`, string(projectID), string(snapshotID), query, limit)
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "postgresfts search", err)
	}
	defer rows.Close()

	var hits []retrieval.SparseHit
	for rows.Next() {
		var chunkID string
		var score float64
		if err := rows.Scan(&chunkID, &score); err != nil {
			return nil, apperr.Wrap(apperr.Internal, "postgresfts scan", err)
		}
		hits = append(hits, retrieval.SparseHit{
			ChunkID: ids.ChunkID(chunkID),
			Score:   score,
		})
	}
	return hits, rows.Err()
}

var (
	_ retrieval.SparseSearchClient = (*Client)(nil)
	_ retrieval.CapabilityReporter = (*Client)(nil)
)
