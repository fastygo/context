// Package postgresvector is the first live VectorStore adapter (ADR-0017).
package postgresvector

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/retrieval"
)

const BackendID = "postgres_pgvector"

// Config configures the pgvector dense store.
type Config struct {
	Collection string
	Dimension  int
	Metric     string // cosine | l2 | ip
}

// Store persists dense vectors in PostgreSQL/pgvector.
type Store struct {
	pool *pgxpool.Pool
	cfg  Config
}

// Open connects and validates config. Call EnsureSchema before Upsert/Search.
func Open(ctx context.Context, dsn string, cfg Config) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, apperr.New(apperr.Validation, "postgresvector: dsn required")
	}
	if cfg.Collection == "" {
		cfg.Collection = config.DefaultVectorCollection
	}
	if cfg.Dimension <= 0 {
		cfg.Dimension = config.DefaultEmbeddingDimension
	}
	if cfg.Metric == "" {
		cfg.Metric = config.DefaultVectorMetric
	}
	switch cfg.Metric {
	case "cosine", "l2", "ip":
	default:
		return nil, apperr.New(apperr.Validation, "postgresvector: metric must be cosine|l2|ip")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, apperr.Wrap(apperr.Unavailable, "postgresvector connect", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, apperr.Wrap(apperr.Unavailable, "postgresvector ping", err)
	}
	return &Store{pool: pool, cfg: cfg}, nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

// Capabilities reports server-side filter support for planners.
func (s *Store) Capabilities() retrieval.BackendCapabilities {
	return retrieval.BackendCapabilities{
		BackendID:                BackendID,
		Kind:                     string(config.StoreKindPostgresVector),
		SupportsProjectFilter:    true,
		SupportsSnapshotFilter:   true,
		SupportsTemporalFilter:   false, // temporal remains client-side via chunk index
		SupportsMetadataFilter:   false, // lexicon/language filters remain client-side
		SupportsPayloadNamespace: true,
		Dimension:                s.cfg.Dimension,
		MaxDimension:             s.cfg.Dimension,
		Metrics:                  []string{s.cfg.Metric},
		NamespaceModel:           "shared_collection_payload_filter",
		ManagedService:           false,
	}
}

// EnsureSchema creates the extension and dense table for the configured dimension.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "postgresvector create extension", err)
	}
	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  collection text NOT NULL,
  project_id text NOT NULL,
  snapshot_id text NOT NULL,
  chunk_id text NOT NULL,
  embedding_version text NOT NULL,
  chunker_version text NOT NULL DEFAULT '',
  morph_version text NOT NULL DEFAULT '',
  context_ref text NOT NULL DEFAULT '',
  language text NOT NULL DEFAULT '',
  span_start bigint NOT NULL DEFAULT 0,
  span_end bigint NOT NULL DEFAULT 0,
  embedding vector(%d) NOT NULL,
  PRIMARY KEY (collection, project_id, snapshot_id, chunk_id, embedding_version)
)`, s.tableName(), s.cfg.Dimension)
	if _, err := s.pool.Exec(ctx, ddl); err != nil {
		return apperr.Wrap(apperr.Internal, "postgresvector create table", err)
	}
	return nil
}

// tableName is dimension-scoped so changing embed dim does not collide with
// CREATE TABLE IF NOT EXISTS on a fixed vector(N) column (Chunk 16).
func (s *Store) tableName() string {
	return fmt.Sprintf("context_dense_vectors_d%d", s.cfg.Dimension)
}

func (s *Store) Upsert(ctx context.Context, ns indexing.VectorNamespace, points []retrieval.VectorPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ns.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "vector_namespace", err)
	}
	if ns.EmbeddingVersion == "" {
		return apperr.New(apperr.Validation, "embedding_version required")
	}
	collection := firstNonEmpty(ns.Name, s.cfg.Collection)
	batch := &pgx.Batch{}
	for _, p := range points {
		if p.ProjectID != ns.ProjectID || p.SnapshotID != ns.SnapshotID {
			return apperr.New(apperr.Validation, "vector point project/snapshot must match namespace")
		}
		if err := p.ProjectID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "project_id", err)
		}
		if err := p.SnapshotID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "snapshot_id", err)
		}
		if err := p.ChunkID.Validate(); err != nil {
			return apperr.Wrap(apperr.Validation, "chunk_id", err)
		}
		if p.EmbeddingVersion == "" {
			return apperr.New(apperr.Validation, "embedding_version required")
		}
		if p.EmbeddingVersion != ns.EmbeddingVersion {
			return apperr.New(apperr.Validation, "embedding_version must match namespace")
		}
		if len(p.Vector) != s.cfg.Dimension {
			return apperr.New(apperr.Validation, fmt.Sprintf(
				"vector dimension %d != configured %d", len(p.Vector), s.cfg.Dimension))
		}
		batch.Queue(fmt.Sprintf(`
INSERT INTO %s (
  collection, project_id, snapshot_id, chunk_id, embedding_version,
  chunker_version, morph_version, context_ref, language, span_start, span_end, embedding
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::vector)
ON CONFLICT (collection, project_id, snapshot_id, chunk_id, embedding_version)
DO UPDATE SET
  chunker_version = EXCLUDED.chunker_version,
  morph_version = EXCLUDED.morph_version,
  context_ref = EXCLUDED.context_ref,
  language = EXCLUDED.language,
  span_start = EXCLUDED.span_start,
  span_end = EXCLUDED.span_end,
  embedding = EXCLUDED.embedding
`, s.tableName()), collection, string(p.ProjectID), string(p.SnapshotID), string(p.ChunkID), p.EmbeddingVersion,
			p.ChunkerVersion, p.MorphVersion, string(p.ContextRef), p.Language,
			p.Span.Start, p.Span.End, formatVector(p.Vector))
	}
	if batch.Len() == 0 {
		return nil
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return apperr.Wrap(apperr.Internal, "postgresvector upsert", err)
		}
	}
	return nil
}

func (s *Store) Search(ctx context.Context, ns indexing.VectorNamespace, vector []float32, limit int) ([]retrieval.VectorHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ns.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "vector_namespace", err)
	}
	if ns.EmbeddingVersion == "" {
		return nil, apperr.New(apperr.Validation, "embedding_version required")
	}
	if err := ns.ProjectID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := ns.SnapshotID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "snapshot_id", err)
	}
	if len(vector) != s.cfg.Dimension {
		return nil, apperr.New(apperr.Validation, fmt.Sprintf(
			"query vector dimension %d != configured %d", len(vector), s.cfg.Dimension))
	}
	if limit <= 0 {
		limit = 20
	}
	collection := firstNonEmpty(ns.Name, s.cfg.Collection)
	op, scoreExpr := distanceSQL(s.cfg.Metric)
	q := fmt.Sprintf(`
SELECT chunk_id, embedding_version, chunker_version, morph_version, context_ref, snapshot_id,
       %s AS score
FROM %s
WHERE collection = $1
  AND project_id = $2
  AND snapshot_id = $3
  AND embedding_version = $4
ORDER BY embedding %s $5::vector
LIMIT $6
`, scoreExpr, s.tableName(), op)

	rows, err := s.pool.Query(ctx, q,
		collection, string(ns.ProjectID), string(ns.SnapshotID), ns.EmbeddingVersion,
		formatVector(vector), limit)
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "postgresvector search", err)
	}
	defer rows.Close()

	var hits []retrieval.VectorHit
	for rows.Next() {
		var (
			chunkID, embVer, chunkerVer, morphVer, contextRef, snapshotID string
			score                                                         float64
		)
		if err := rows.Scan(&chunkID, &embVer, &chunkerVer, &morphVer, &contextRef, &snapshotID, &score); err != nil {
			return nil, apperr.Wrap(apperr.Internal, "postgresvector scan", err)
		}
		hits = append(hits, retrieval.VectorHit{
			ChunkID:          ids.ChunkID(chunkID),
			Score:            score,
			EmbeddingVersion: embVer,
			ChunkerVersion:   chunkerVer,
			MorphVersion:     morphVer,
			ContextRef:       ids.ContextRefID(contextRef),
			SnapshotID:       ids.SnapshotID(snapshotID),
		})
	}
	return hits, rows.Err()
}

func distanceSQL(metric string) (op, scoreExpr string) {
	switch metric {
	case "l2":
		return "<->", "(-(embedding <-> $5::vector))"
	case "ip":
		return "<#>", "(-(embedding <#> $5::vector))"
	default: // cosine
		return "<=>", "(1 - (embedding <=> $5::vector))"
	}
}

func formatVector(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

var (
	_ retrieval.VectorStore         = (*Store)(nil)
	_ retrieval.CapabilityReporter  = (*Store)(nil)
)
