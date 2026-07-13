// Package postgres implements durable MetadataStore on PostgreSQL (Chunk 11).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/tools"
	"github.com/fastygo/context/internal/tracing"
)

// Store is a PostgreSQL MetadataStore / ArtifactMetaStore / DocumentStore.
type Store struct {
	pool *pgxpool.Pool
}

type dbConn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// Open connects to PostgreSQL and applies metadata migrations.
func Open(ctx context.Context, dsn string) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, apperr.New(apperr.Validation, "postgres metadata: dsn required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, apperr.Wrap(apperr.Unavailable, "postgres metadata connect", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, apperr.Wrap(apperr.Unavailable, "postgres metadata ping", err)
	}
	s := &Store{pool: pool}
	if err := s.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the pool.
func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

// WithTx runs fn inside a SQL transaction. Nested WithTx reuses the same tx.
func (s *Store) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "begin tx", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ctx = context.WithValue(ctx, txKey{}, tx)
	if err := fn(ctx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return apperr.Wrap(apperr.Internal, "commit tx", err)
	}
	return nil
}

func (s *Store) conn(ctx context.Context) dbConn {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return s.pool
}

func (s *Store) PutProject(ctx context.Context, project corpus.Project) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := project.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "project", err)
	}
	_, err := s.conn(ctx).Exec(ctx, `
INSERT INTO projects (id, name, active_snapshot_id, updated_at)
VALUES ($1,$2,$3,now())
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  active_snapshot_id = EXCLUDED.active_snapshot_id,
  updated_at = now()
`, string(project.ID), project.Name, string(project.ActiveSnapshotID))
	return wrapDB(err, "put project")
}

func (s *Store) GetProject(ctx context.Context, id ids.ProjectID) (corpus.Project, error) {
	if err := ctx.Err(); err != nil {
		return corpus.Project{}, err
	}
	var p corpus.Project
	var active string
	err := s.conn(ctx).QueryRow(ctx, `
SELECT id, name, active_snapshot_id FROM projects WHERE id = $1
`, string(id)).Scan(&p.ID, &p.Name, &active)
	if err != nil {
		return corpus.Project{}, mapNotFound(err, "project not found")
	}
	p.ActiveSnapshotID = ids.SnapshotID(active)
	return p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]corpus.Project, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.conn(ctx).Query(ctx, `SELECT id, name, active_snapshot_id FROM projects ORDER BY id`)
	if err != nil {
		return nil, wrapDB(err, "list projects")
	}
	defer rows.Close()
	var out []corpus.Project
	for rows.Next() {
		var p corpus.Project
		var active string
		if err := rows.Scan(&p.ID, &p.Name, &active); err != nil {
			return nil, wrapDB(err, "scan project")
		}
		p.ActiveSnapshotID = ids.SnapshotID(active)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) PutSource(ctx context.Context, source corpus.Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := source.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "source", err)
	}
	if err := s.requireProject(ctx, source.ProjectID); err != nil {
		return err
	}
	ts, te, tb, ing := temporalCols(source.TemporalMetadata)
	_, err := s.conn(ctx).Exec(ctx, `
INSERT INTO sources (
  project_id, source_id, source_type, path_key, uri, trust_level, media_type, checksum,
  temporal_start, temporal_end, temporal_basis, ingested_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (project_id, source_id) DO UPDATE SET
  source_type = EXCLUDED.source_type,
  path_key = EXCLUDED.path_key,
  uri = EXCLUDED.uri,
  trust_level = EXCLUDED.trust_level,
  media_type = EXCLUDED.media_type,
  checksum = EXCLUDED.checksum,
  temporal_start = EXCLUDED.temporal_start,
  temporal_end = EXCLUDED.temporal_end,
  temporal_basis = EXCLUDED.temporal_basis,
  ingested_at = EXCLUDED.ingested_at
`, string(source.ProjectID), string(source.ID), string(source.Type), source.PathKey, source.URI,
		string(source.TrustLevel), source.MediaType, string(source.Checksum),
		ts, te, tb, ing)
	return wrapDB(err, "put source")
}

func (s *Store) GetSource(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID) (corpus.Source, error) {
	if err := ctx.Err(); err != nil {
		return corpus.Source{}, err
	}
	var src corpus.Source
	var trust, checksum string
	var ts, te, ing *time.Time
	var tb *string
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, source_id, source_type, path_key, uri, trust_level, media_type, checksum,
       temporal_start, temporal_end, temporal_basis, ingested_at
FROM sources WHERE project_id = $1 AND source_id = $2
`, string(projectID), string(sourceID)).Scan(
		&src.ProjectID, &src.ID, &src.Type, &src.PathKey, &src.URI, &trust, &src.MediaType, &checksum,
		&ts, &te, &tb, &ing)
	if err != nil {
		return corpus.Source{}, mapNotFound(err, "source not found")
	}
	src.TrustLevel = foundation.TrustLevel(trust)
	src.Checksum = foundation.ChecksumHex(checksum)
	src.TemporalMetadata = temporalFromCols(ts, te, tb, ing)
	return src, nil
}

func (s *Store) ListSources(ctx context.Context, projectID ids.ProjectID) ([]corpus.Source, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.conn(ctx).Query(ctx, `
SELECT project_id, source_id, source_type, path_key, uri, trust_level, media_type, checksum,
       temporal_start, temporal_end, temporal_basis, ingested_at
FROM sources WHERE project_id = $1 ORDER BY source_id
`, string(projectID))
	if err != nil {
		return nil, wrapDB(err, "list sources")
	}
	defer rows.Close()
	var out []corpus.Source
	for rows.Next() {
		var src corpus.Source
		var trust, checksum string
		var ts, te, ing *time.Time
		var tb *string
		if err := rows.Scan(
			&src.ProjectID, &src.ID, &src.Type, &src.PathKey, &src.URI, &trust, &src.MediaType, &checksum,
			&ts, &te, &tb, &ing); err != nil {
			return nil, wrapDB(err, "scan source")
		}
		src.TrustLevel = foundation.TrustLevel(trust)
		src.Checksum = foundation.ChecksumHex(checksum)
		src.TemporalMetadata = temporalFromCols(ts, te, tb, ing)
		out = append(out, src)
	}
	return out, rows.Err()
}

func (s *Store) PutChunk(ctx context.Context, chunk corpus.Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := chunk.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "chunk", err)
	}
	if err := s.requireProject(ctx, chunk.ProjectID); err != nil {
		return err
	}
	ts, te, tb, ing := temporalCols(chunk.TemporalMetadata)
	_, err := s.conn(ctx).Exec(ctx, `
INSERT INTO chunks (
  project_id, chunk_id, source_id, artifact_id, snapshot_id, chunker_version,
  span_start, span_end, text_checksum, chunk_hash, language, embedding_version, sparse_version,
  temporal_start, temporal_end, temporal_basis, ingested_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
ON CONFLICT (project_id, chunk_id) DO UPDATE SET
  source_id = EXCLUDED.source_id,
  artifact_id = EXCLUDED.artifact_id,
  snapshot_id = EXCLUDED.snapshot_id,
  chunker_version = EXCLUDED.chunker_version,
  span_start = EXCLUDED.span_start,
  span_end = EXCLUDED.span_end,
  text_checksum = EXCLUDED.text_checksum,
  chunk_hash = EXCLUDED.chunk_hash,
  language = EXCLUDED.language,
  embedding_version = EXCLUDED.embedding_version,
  sparse_version = EXCLUDED.sparse_version,
  temporal_start = EXCLUDED.temporal_start,
  temporal_end = EXCLUDED.temporal_end,
  temporal_basis = EXCLUDED.temporal_basis,
  ingested_at = EXCLUDED.ingested_at
`, string(chunk.ProjectID), string(chunk.ID), string(chunk.SourceID), string(chunk.ArtifactID),
		string(chunk.SnapshotID), chunk.ChunkerVersion, int64(chunk.Span.Start), int64(chunk.Span.End),
		string(chunk.TextChecksum), string(chunk.ChunkHash), chunk.Language, chunk.EmbeddingVersion, chunk.SparseVersion,
		ts, te, tb, ing)
	return wrapDB(err, "put chunk")
}

func (s *Store) GetChunk(ctx context.Context, projectID ids.ProjectID, chunkID ids.ChunkID) (corpus.Chunk, error) {
	if err := ctx.Err(); err != nil {
		return corpus.Chunk{}, err
	}
	var ch corpus.Chunk
	var textSum, chunkHash string
	var spanStart, spanEnd int64
	var ts, te, ing *time.Time
	var tb *string
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, chunk_id, source_id, artifact_id, snapshot_id, chunker_version,
       span_start, span_end, text_checksum, chunk_hash, language, embedding_version, sparse_version,
       temporal_start, temporal_end, temporal_basis, ingested_at
FROM chunks WHERE project_id = $1 AND chunk_id = $2
`, string(projectID), string(chunkID)).Scan(
		&ch.ProjectID, &ch.ID, &ch.SourceID, &ch.ArtifactID, &ch.SnapshotID, &ch.ChunkerVersion,
		&spanStart, &spanEnd, &textSum, &chunkHash, &ch.Language, &ch.EmbeddingVersion, &ch.SparseVersion,
		&ts, &te, &tb, &ing)
	if err != nil {
		return corpus.Chunk{}, mapNotFound(err, "chunk not found")
	}
	ch.Span = foundation.ByteSpan{Start: uint64(spanStart), End: uint64(spanEnd)}
	ch.TextChecksum = foundation.ChecksumHex(textSum)
	ch.ChunkHash = foundation.ChecksumHex(chunkHash)
	ch.TemporalMetadata = temporalFromCols(ts, te, tb, ing)
	return ch, nil
}

func (s *Store) ListChunks(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) ([]corpus.Chunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.conn(ctx).Query(ctx, `
SELECT project_id, chunk_id, source_id, artifact_id, snapshot_id, chunker_version,
       span_start, span_end, text_checksum, chunk_hash, language, embedding_version, sparse_version,
       temporal_start, temporal_end, temporal_basis, ingested_at
FROM chunks WHERE project_id = $1 AND snapshot_id = $2 ORDER BY chunk_id
`, string(projectID), string(snapshotID))
	if err != nil {
		return nil, wrapDB(err, "list chunks")
	}
	defer rows.Close()
	var out []corpus.Chunk
	for rows.Next() {
		var ch corpus.Chunk
		var textSum, chunkHash string
		var spanStart, spanEnd int64
		var ts, te, ing *time.Time
		var tb *string
		if err := rows.Scan(
			&ch.ProjectID, &ch.ID, &ch.SourceID, &ch.ArtifactID, &ch.SnapshotID, &ch.ChunkerVersion,
			&spanStart, &spanEnd, &textSum, &chunkHash, &ch.Language, &ch.EmbeddingVersion, &ch.SparseVersion,
			&ts, &te, &tb, &ing); err != nil {
			return nil, wrapDB(err, "scan chunk")
		}
		ch.Span = foundation.ByteSpan{Start: uint64(spanStart), End: uint64(spanEnd)}
		ch.TextChecksum = foundation.ChecksumHex(textSum)
		ch.ChunkHash = foundation.ChecksumHex(chunkHash)
		ch.TemporalMetadata = temporalFromCols(ts, te, tb, ing)
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *Store) PutSnapshot(ctx context.Context, snapshot indexing.IndexSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := snapshot.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "snapshot", err)
	}
	if err := s.requireProject(ctx, snapshot.ProjectID); err != nil {
		return err
	}
	sparseJSON, err := json.Marshal(snapshot.SparseIndexRef)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "sparse_index_ref", err)
	}
	nsJSON, err := json.Marshal(snapshot.VectorNamespace)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "vector_namespace", err)
	}
	_, err = s.conn(ctx).Exec(ctx, `
INSERT INTO index_snapshots (
  project_id, snapshot_id, parent_snapshot_id, status, source_merkle_root, chunk_set_hash,
  source_merkle_algo, chunk_set_merkle_algo, parser_version, chunker_version, embed_model_version,
  morph_version, sparse_index_ref, vector_namespace, dense_enabled, sparse_enabled, failure_reason
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
ON CONFLICT (project_id, snapshot_id) DO UPDATE SET
  parent_snapshot_id = EXCLUDED.parent_snapshot_id,
  status = EXCLUDED.status,
  source_merkle_root = EXCLUDED.source_merkle_root,
  chunk_set_hash = EXCLUDED.chunk_set_hash,
  source_merkle_algo = EXCLUDED.source_merkle_algo,
  chunk_set_merkle_algo = EXCLUDED.chunk_set_merkle_algo,
  parser_version = EXCLUDED.parser_version,
  chunker_version = EXCLUDED.chunker_version,
  embed_model_version = EXCLUDED.embed_model_version,
  morph_version = EXCLUDED.morph_version,
  sparse_index_ref = EXCLUDED.sparse_index_ref,
  vector_namespace = EXCLUDED.vector_namespace,
  dense_enabled = EXCLUDED.dense_enabled,
  sparse_enabled = EXCLUDED.sparse_enabled,
  failure_reason = EXCLUDED.failure_reason
`, string(snapshot.ProjectID), string(snapshot.ID), string(snapshot.ParentSnapshotID), string(snapshot.Status),
		string(snapshot.SourceMerkleRoot), string(snapshot.ChunkSetHash), snapshot.SourceMerkleAlgo, snapshot.ChunkSetMerkleAlgo,
		snapshot.ParserVersion, snapshot.ChunkerVersion, snapshot.EmbedModelVersion, snapshot.MorphVersion,
		sparseJSON, nsJSON, snapshot.DenseEnabled, snapshot.SparseEnabled, snapshot.FailureReason)
	return wrapDB(err, "put snapshot")
}

func (s *Store) GetSnapshot(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) (indexing.IndexSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return indexing.IndexSnapshot{}, err
	}
	var snap indexing.IndexSnapshot
	var status, root, chunkHash string
	var sparseJSON, nsJSON []byte
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, snapshot_id, parent_snapshot_id, status, source_merkle_root, chunk_set_hash,
       source_merkle_algo, chunk_set_merkle_algo, parser_version, chunker_version, embed_model_version,
       morph_version, sparse_index_ref, vector_namespace, dense_enabled, sparse_enabled, failure_reason
FROM index_snapshots WHERE project_id = $1 AND snapshot_id = $2
`, string(projectID), string(snapshotID)).Scan(
		&snap.ProjectID, &snap.ID, &snap.ParentSnapshotID, &status, &root, &chunkHash,
		&snap.SourceMerkleAlgo, &snap.ChunkSetMerkleAlgo, &snap.ParserVersion, &snap.ChunkerVersion, &snap.EmbedModelVersion,
		&snap.MorphVersion, &sparseJSON, &nsJSON, &snap.DenseEnabled, &snap.SparseEnabled, &snap.FailureReason)
	if err != nil {
		return indexing.IndexSnapshot{}, mapNotFound(err, "snapshot not found")
	}
	snap.Status = foundation.SnapshotStatus(status)
	snap.SourceMerkleRoot = foundation.ChecksumHex(root)
	snap.ChunkSetHash = foundation.ChecksumHex(chunkHash)
	_ = json.Unmarshal(sparseJSON, &snap.SparseIndexRef)
	_ = json.Unmarshal(nsJSON, &snap.VectorNamespace)
	return snap, nil
}

func (s *Store) SetActiveSnapshot(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) error {
	return s.WithTx(ctx, func(ctx context.Context) error {
		project, err := s.GetProject(ctx, projectID)
		if err != nil {
			return err
		}
		snap, err := s.GetSnapshot(ctx, projectID, snapshotID)
		if err != nil {
			return err
		}
		if snap.Status != foundation.SnapshotReady {
			return apperr.New(apperr.Conflict, "active snapshot must be ready")
		}
		if project.ActiveSnapshotID != "" && project.ActiveSnapshotID != snapshotID {
			_, err := s.conn(ctx).Exec(ctx, `
UPDATE index_snapshots SET status = $1
WHERE project_id = $2 AND snapshot_id = $3 AND status = $4
`, string(foundation.SnapshotSuperseded), string(projectID), string(project.ActiveSnapshotID), string(foundation.SnapshotReady))
			if err != nil {
				return wrapDB(err, "supersede snapshot")
			}
		}
		project.ActiveSnapshotID = snapshotID
		return s.PutProject(ctx, project)
	})
}

func (s *Store) PutPack(ctx context.Context, pack retrieval.ContextPack) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := pack.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "context_pack", err)
	}
	if err := s.requireProject(ctx, pack.ProjectID); err != nil {
		return err
	}
	payload, err := json.Marshal(pack)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "pack json", err)
	}
	_, err = s.conn(ctx).Exec(ctx, `
INSERT INTO context_packs (project_id, pack_id, task_id, retrieval_plan_id, purpose, checksum, payload)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (project_id, pack_id) DO UPDATE SET
  task_id = EXCLUDED.task_id,
  retrieval_plan_id = EXCLUDED.retrieval_plan_id,
  purpose = EXCLUDED.purpose,
  checksum = EXCLUDED.checksum,
  payload = EXCLUDED.payload
`, string(pack.ProjectID), string(pack.ID), string(pack.TaskID), string(pack.RetrievalPlanID),
		pack.Purpose, string(pack.Checksum), payload)
	return wrapDB(err, "put pack")
}

func (s *Store) GetPack(ctx context.Context, projectID ids.ProjectID, packID ids.PackID) (retrieval.ContextPack, error) {
	if err := ctx.Err(); err != nil {
		return retrieval.ContextPack{}, err
	}
	var payload []byte
	err := s.conn(ctx).QueryRow(ctx, `
SELECT payload FROM context_packs WHERE project_id = $1 AND pack_id = $2
`, string(projectID), string(packID)).Scan(&payload)
	if err != nil {
		return retrieval.ContextPack{}, mapNotFound(err, "context pack not found")
	}
	var pack retrieval.ContextPack
	if err := json.Unmarshal(payload, &pack); err != nil {
		return retrieval.ContextPack{}, apperr.Wrap(apperr.Internal, "pack decode", err)
	}
	return pack, nil
}

func (s *Store) PutRun(ctx context.Context, run agentruntime.AgentRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := run.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "agent_run", err)
	}
	if err := s.requireProject(ctx, run.ProjectID); err != nil {
		return err
	}
	_, err := s.conn(ctx).Exec(ctx, `
INSERT INTO agent_runs (
  project_id, run_id, task_id, mode, status, focus_id, policy_id, pack_id, parent_run_id,
  owner, created_at, updated_at, error
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (project_id, run_id) DO UPDATE SET
  task_id = EXCLUDED.task_id,
  mode = EXCLUDED.mode,
  status = EXCLUDED.status,
  focus_id = EXCLUDED.focus_id,
  policy_id = EXCLUDED.policy_id,
  pack_id = EXCLUDED.pack_id,
  parent_run_id = EXCLUDED.parent_run_id,
  owner = EXCLUDED.owner,
  created_at = EXCLUDED.created_at,
  updated_at = EXCLUDED.updated_at,
  error = EXCLUDED.error
`, string(run.ProjectID), string(run.ID), string(run.TaskID), string(run.Mode), string(run.Status),
		string(run.FocusID), string(run.PolicyID), string(run.PackID), string(run.ParentRunID),
		run.Owner, run.CreatedAt.UTC(), run.UpdatedAt.UTC(), run.Error)
	return wrapDB(err, "put run")
}

func (s *Store) GetRun(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) (agentruntime.AgentRun, error) {
	if err := ctx.Err(); err != nil {
		return agentruntime.AgentRun{}, err
	}
	var run agentruntime.AgentRun
	var mode, status string
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, run_id, task_id, mode, status, focus_id, policy_id, pack_id, parent_run_id,
       owner, created_at, updated_at, error
FROM agent_runs WHERE project_id = $1 AND run_id = $2
`, string(projectID), string(runID)).Scan(
		&run.ProjectID, &run.ID, &run.TaskID, &mode, &status, &run.FocusID, &run.PolicyID, &run.PackID, &run.ParentRunID,
		&run.Owner, &run.CreatedAt, &run.UpdatedAt, &run.Error)
	if err != nil {
		return agentruntime.AgentRun{}, mapNotFound(err, "agent run not found")
	}
	run.Mode = agentruntime.RunMode(mode)
	run.Status = agentruntime.RunStatus(status)
	return run, nil
}

func (s *Store) PutToolCall(ctx context.Context, call tools.ToolCall) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := call.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "tool_call", err)
	}
	if err := s.requireProject(ctx, call.ProjectID); err != nil {
		return err
	}
	decision, err := json.Marshal(call.Decision)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "tool decision json", err)
	}
	_, err = s.conn(ctx).Exec(ctx, `
INSERT INTO tool_calls (
  project_id, tool_call_id, run_id, tool_name, input_artifact_id, output_artifact_id,
  status, decision, risk_level, error
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (project_id, tool_call_id) DO UPDATE SET
  run_id = EXCLUDED.run_id,
  tool_name = EXCLUDED.tool_name,
  input_artifact_id = EXCLUDED.input_artifact_id,
  output_artifact_id = EXCLUDED.output_artifact_id,
  status = EXCLUDED.status,
  decision = EXCLUDED.decision,
  risk_level = EXCLUDED.risk_level,
  error = EXCLUDED.error
`, string(call.ProjectID), string(call.ID), string(call.RunID), call.ToolName,
		string(call.InputArtifactID), string(call.OutputArtifactID), call.Status, decision,
		string(call.RiskLevel), call.Error)
	return wrapDB(err, "put tool call")
}

func (s *Store) GetToolCall(ctx context.Context, projectID ids.ProjectID, callID ids.ToolCallID) (tools.ToolCall, error) {
	if err := ctx.Err(); err != nil {
		return tools.ToolCall{}, err
	}
	var call tools.ToolCall
	var decision []byte
	var risk string
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, tool_call_id, run_id, tool_name, input_artifact_id, output_artifact_id,
       status, decision, risk_level, error
FROM tool_calls WHERE project_id = $1 AND tool_call_id = $2
`, string(projectID), string(callID)).Scan(
		&call.ProjectID, &call.ID, &call.RunID, &call.ToolName, &call.InputArtifactID, &call.OutputArtifactID,
		&call.Status, &decision, &risk, &call.Error)
	if err != nil {
		return tools.ToolCall{}, mapNotFound(err, "tool call not found")
	}
	_ = json.Unmarshal(decision, &call.Decision)
	call.RiskLevel = policy.RiskLevel(risk)
	return call, nil
}

func (s *Store) AppendTrace(ctx context.Context, event tracing.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "trace_event", err)
	}
	if err := s.requireProject(ctx, event.ProjectID); err != nil {
		return err
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "trace payload", err)
	}
	if payload == nil {
		payload = []byte("{}")
	}
	_, err = s.conn(ctx).Exec(ctx, `
INSERT INTO trace_events (
  project_id, run_id, event_id, event_type, event_ts, payload,
  analyzer_version, dictionary_version, feature_scheme, query_expansion_ver,
  sense_mapping_version, concept_mapping_ver, attestation_version, snapshot_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
`, string(event.ProjectID), string(event.RunID), string(event.ID), string(event.Type), event.Timestamp.UTC(), payload,
		event.AnalyzerVersion, event.DictionaryVersion, event.FeatureScheme, event.QueryExpansionVer,
		event.SenseMappingVersion, event.ConceptMappingVer, event.AttestationVersion, string(event.SnapshotID))
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.New(apperr.Conflict, "trace event id already exists")
		}
		return wrapDB(err, "append trace")
	}
	return nil
}

func (s *Store) ListTrace(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) ([]tracing.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.conn(ctx).Query(ctx, `
SELECT project_id, run_id, event_id, event_type, event_ts, payload,
       analyzer_version, dictionary_version, feature_scheme, query_expansion_ver,
       sense_mapping_version, concept_mapping_ver, attestation_version, snapshot_id
FROM trace_events WHERE project_id = $1 AND run_id = $2 ORDER BY event_ts, event_id
`, string(projectID), string(runID))
	if err != nil {
		return nil, wrapDB(err, "list trace")
	}
	defer rows.Close()
	var out []tracing.Event
	for rows.Next() {
		var ev tracing.Event
		var typ string
		var payload []byte
		if err := rows.Scan(
			&ev.ProjectID, &ev.RunID, &ev.ID, &typ, &ev.Timestamp, &payload,
			&ev.AnalyzerVersion, &ev.DictionaryVersion, &ev.FeatureScheme, &ev.QueryExpansionVer,
			&ev.SenseMappingVersion, &ev.ConceptMappingVer, &ev.AttestationVersion, &ev.SnapshotID); err != nil {
			return nil, wrapDB(err, "scan trace")
		}
		ev.Type = tracing.EventType(typ)
		_ = json.Unmarshal(payload, &ev.Payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *Store) PutArtifactMeta(ctx context.Context, artifact artifacts.Artifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := artifact.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "artifact", err)
	}
	if err := s.requireProject(ctx, artifact.ProjectID); err != nil {
		return err
	}
	_, err := s.conn(ctx).Exec(ctx, `
INSERT INTO artifacts_meta (
  project_id, artifact_id, source_id, media_type, byte_size, checksum, storage_uri, artifact_type, schema_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (project_id, artifact_id) DO UPDATE SET
  source_id = EXCLUDED.source_id,
  media_type = EXCLUDED.media_type,
  byte_size = EXCLUDED.byte_size,
  checksum = EXCLUDED.checksum,
  storage_uri = EXCLUDED.storage_uri,
  artifact_type = EXCLUDED.artifact_type,
  schema_id = EXCLUDED.schema_id
`, string(artifact.ProjectID), string(artifact.ID), string(artifact.SourceID), artifact.MediaType,
		artifact.ByteSize, string(artifact.Checksum), artifact.StorageURI,
		artifacts.NormalizeType(artifact.ArtifactType), artifact.SchemaID)
	return wrapDB(err, "put artifact meta")
}

func (s *Store) GetArtifactMeta(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) (artifacts.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifacts.Artifact{}, err
	}
	var art artifacts.Artifact
	var checksum string
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, artifact_id, source_id, media_type, byte_size, checksum, storage_uri, artifact_type, schema_id
FROM artifacts_meta WHERE project_id = $1 AND artifact_id = $2
`, string(projectID), string(artifactID)).Scan(
		&art.ProjectID, &art.ID, &art.SourceID, &art.MediaType, &art.ByteSize, &checksum,
		&art.StorageURI, &art.ArtifactType, &art.SchemaID)
	if err != nil {
		return artifacts.Artifact{}, mapNotFound(err, "artifact meta not found")
	}
	art.Checksum = foundation.ChecksumHex(checksum)
	return art, nil
}

func (s *Store) ListArtifacts(ctx context.Context, projectID ids.ProjectID) ([]artifacts.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.conn(ctx).Query(ctx, `
SELECT project_id, artifact_id, source_id, media_type, byte_size, checksum, storage_uri, artifact_type, schema_id
FROM artifacts_meta WHERE project_id = $1 ORDER BY artifact_id
`, string(projectID))
	if err != nil {
		return nil, wrapDB(err, "list artifacts")
	}
	defer rows.Close()
	var out []artifacts.Artifact
	for rows.Next() {
		var art artifacts.Artifact
		var checksum string
		if err := rows.Scan(
			&art.ProjectID, &art.ID, &art.SourceID, &art.MediaType, &art.ByteSize, &checksum,
			&art.StorageURI, &art.ArtifactType, &art.SchemaID); err != nil {
			return nil, wrapDB(err, "scan artifact")
		}
		art.Checksum = foundation.ChecksumHex(checksum)
		out = append(out, art)
	}
	return out, rows.Err()
}

func (s *Store) PutArtifactLineage(ctx context.Context, lineage artifacts.ArtifactLineage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := lineage.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "artifact_lineage", err)
	}
	if err := s.requireProject(ctx, lineage.ProjectID); err != nil {
		return err
	}
	if _, err := s.GetArtifactMeta(ctx, lineage.ProjectID, lineage.OutputArtifactID); err != nil {
		return err
	}
	for _, inputID := range lineage.InputArtifactIDs {
		if _, err := s.GetArtifactMeta(ctx, lineage.ProjectID, inputID); err != nil {
			return apperr.New(apperr.NotFound, "input artifact metadata not found")
		}
	}
	inputs, err := json.Marshal(lineage.InputArtifactIDs)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "lineage inputs", err)
	}
	refs, err := json.Marshal(lineage.SourceRefs)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "lineage source refs", err)
	}
	_, err = s.conn(ctx).Exec(ctx, `
INSERT INTO artifact_lineage (
  project_id, output_artifact_id, input_artifact_ids, source_refs, context_pack_id, agent_run_id,
  tool_call_id, generator_id, generator_version, transformation_kind, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
`, string(lineage.ProjectID), string(lineage.OutputArtifactID), inputs, refs,
		string(lineage.ContextPackID), string(lineage.AgentRunID), string(lineage.ToolCallID),
		lineage.GeneratorID, lineage.GeneratorVersion, lineage.TransformationKind, lineage.CreatedAt.UTC())
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.New(apperr.Conflict, "artifact lineage already exists")
		}
		return wrapDB(err, "put lineage")
	}
	return nil
}

func (s *Store) GetArtifactLineage(ctx context.Context, projectID ids.ProjectID, outputArtifactID ids.ArtifactID) (artifacts.ArtifactLineage, error) {
	if err := ctx.Err(); err != nil {
		return artifacts.ArtifactLineage{}, err
	}
	var lineage artifacts.ArtifactLineage
	var inputs, refs []byte
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, output_artifact_id, input_artifact_ids, source_refs, context_pack_id, agent_run_id,
       tool_call_id, generator_id, generator_version, transformation_kind, created_at
FROM artifact_lineage WHERE project_id = $1 AND output_artifact_id = $2
`, string(projectID), string(outputArtifactID)).Scan(
		&lineage.ProjectID, &lineage.OutputArtifactID, &inputs, &refs, &lineage.ContextPackID, &lineage.AgentRunID,
		&lineage.ToolCallID, &lineage.GeneratorID, &lineage.GeneratorVersion, &lineage.TransformationKind, &lineage.CreatedAt)
	if err != nil {
		return artifacts.ArtifactLineage{}, mapNotFound(err, "artifact lineage not found")
	}
	_ = json.Unmarshal(inputs, &lineage.InputArtifactIDs)
	_ = json.Unmarshal(refs, &lineage.SourceRefs)
	return lineage, nil
}

func (s *Store) ListArtifactLineage(ctx context.Context, projectID ids.ProjectID) ([]artifacts.ArtifactLineage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.conn(ctx).Query(ctx, `
SELECT project_id, output_artifact_id, input_artifact_ids, source_refs, context_pack_id, agent_run_id,
       tool_call_id, generator_id, generator_version, transformation_kind, created_at
FROM artifact_lineage WHERE project_id = $1 ORDER BY output_artifact_id
`, string(projectID))
	if err != nil {
		return nil, wrapDB(err, "list lineage")
	}
	defer rows.Close()
	var out []artifacts.ArtifactLineage
	for rows.Next() {
		var lineage artifacts.ArtifactLineage
		var inputs, refs []byte
		if err := rows.Scan(
			&lineage.ProjectID, &lineage.OutputArtifactID, &inputs, &refs, &lineage.ContextPackID, &lineage.AgentRunID,
			&lineage.ToolCallID, &lineage.GeneratorID, &lineage.GeneratorVersion, &lineage.TransformationKind, &lineage.CreatedAt); err != nil {
			return nil, wrapDB(err, "scan lineage")
		}
		_ = json.Unmarshal(inputs, &lineage.InputArtifactIDs)
		_ = json.Unmarshal(refs, &lineage.SourceRefs)
		out = append(out, lineage)
	}
	return out, rows.Err()
}

func (s *Store) PutDocument(ctx context.Context, doc storage.MetaDocument) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := doc.ProjectID.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if strings.TrimSpace(string(doc.Kind)) == "" || strings.TrimSpace(doc.ID) == "" {
		return apperr.New(apperr.Validation, "document kind and id required")
	}
	if len(doc.Payload) == 0 {
		return apperr.New(apperr.Validation, "document payload required")
	}
	if !json.Valid(doc.Payload) {
		return apperr.New(apperr.Validation, "document payload must be JSON")
	}
	if err := s.requireProject(ctx, doc.ProjectID); err != nil {
		return err
	}
	_, err := s.conn(ctx).Exec(ctx, `
INSERT INTO meta_documents (
  project_id, kind, document_id, language, lexeme_id, sense_id, concept_id, region, register,
  time_period, lexicon_source_id, source_authority, analyzer_version, dictionary_version,
  snapshot_id, chunk_id, payload, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,now())
ON CONFLICT (project_id, kind, document_id) DO UPDATE SET
  language = EXCLUDED.language,
  lexeme_id = EXCLUDED.lexeme_id,
  sense_id = EXCLUDED.sense_id,
  concept_id = EXCLUDED.concept_id,
  region = EXCLUDED.region,
  register = EXCLUDED.register,
  time_period = EXCLUDED.time_period,
  lexicon_source_id = EXCLUDED.lexicon_source_id,
  source_authority = EXCLUDED.source_authority,
  analyzer_version = EXCLUDED.analyzer_version,
  dictionary_version = EXCLUDED.dictionary_version,
  snapshot_id = EXCLUDED.snapshot_id,
  chunk_id = EXCLUDED.chunk_id,
  payload = EXCLUDED.payload,
  updated_at = now()
`, string(doc.ProjectID), string(doc.Kind), doc.ID, doc.Language, doc.LexemeID, doc.SenseID, doc.ConceptID,
		doc.Region, doc.Register, doc.TimePeriod, doc.LexiconSourceID, doc.SourceAuthority,
		doc.AnalyzerVersion, doc.DictionaryVersion, string(doc.SnapshotID), string(doc.ChunkID), doc.Payload)
	return wrapDB(err, "put document")
}

func (s *Store) GetDocument(ctx context.Context, projectID ids.ProjectID, kind storage.DocumentKind, id string) (storage.MetaDocument, error) {
	if err := ctx.Err(); err != nil {
		return storage.MetaDocument{}, err
	}
	var doc storage.MetaDocument
	var kindStr string
	err := s.conn(ctx).QueryRow(ctx, `
SELECT project_id, kind, document_id, language, lexeme_id, sense_id, concept_id, region, register,
       time_period, lexicon_source_id, source_authority, analyzer_version, dictionary_version,
       snapshot_id, chunk_id, payload
FROM meta_documents WHERE project_id = $1 AND kind = $2 AND document_id = $3
`, string(projectID), string(kind), id).Scan(
		&doc.ProjectID, &kindStr, &doc.ID, &doc.Language, &doc.LexemeID, &doc.SenseID, &doc.ConceptID,
		&doc.Region, &doc.Register, &doc.TimePeriod, &doc.LexiconSourceID, &doc.SourceAuthority,
		&doc.AnalyzerVersion, &doc.DictionaryVersion, &doc.SnapshotID, &doc.ChunkID, &doc.Payload)
	if err != nil {
		return storage.MetaDocument{}, mapNotFound(err, "document not found")
	}
	doc.Kind = storage.DocumentKind(kindStr)
	return doc, nil
}

func (s *Store) ListDocuments(ctx context.Context, projectID ids.ProjectID, kind storage.DocumentKind) ([]storage.MetaDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.conn(ctx).Query(ctx, `
SELECT project_id, kind, document_id, language, lexeme_id, sense_id, concept_id, region, register,
       time_period, lexicon_source_id, source_authority, analyzer_version, dictionary_version,
       snapshot_id, chunk_id, payload
FROM meta_documents WHERE project_id = $1 AND kind = $2 ORDER BY document_id
`, string(projectID), string(kind))
	if err != nil {
		return nil, wrapDB(err, "list documents")
	}
	defer rows.Close()
	var out []storage.MetaDocument
	for rows.Next() {
		var doc storage.MetaDocument
		var kindStr string
		if err := rows.Scan(
			&doc.ProjectID, &kindStr, &doc.ID, &doc.Language, &doc.LexemeID, &doc.SenseID, &doc.ConceptID,
			&doc.Region, &doc.Register, &doc.TimePeriod, &doc.LexiconSourceID, &doc.SourceAuthority,
			&doc.AnalyzerVersion, &doc.DictionaryVersion, &doc.SnapshotID, &doc.ChunkID, &doc.Payload); err != nil {
			return nil, wrapDB(err, "scan document")
		}
		doc.Kind = storage.DocumentKind(kindStr)
		out = append(out, doc)
	}
	return out, rows.Err()
}

func (s *Store) requireProject(ctx context.Context, id ids.ProjectID) error {
	_, err := s.GetProject(ctx, id)
	return err
}

func temporalCols(m *corpus.TemporalMetadata) (start, end any, basis any, ingested any) {
	if m == nil {
		return nil, nil, nil, nil
	}
	return m.Range.Start.UTC(), m.Range.End.UTC(), string(m.Range.Basis), m.IngestedAt.UTC()
}

func temporalFromCols(start, end *time.Time, basis *string, ingested *time.Time) *corpus.TemporalMetadata {
	if start == nil || end == nil || ingested == nil || basis == nil || *basis == "" {
		return nil
	}
	return &corpus.TemporalMetadata{
		Range: corpus.TemporalRange{
			Start: start.UTC(),
			End:   end.UTC(),
			Basis: corpus.TimeBasis(*basis),
		},
		IngestedAt: ingested.UTC(),
	}
}

func wrapDB(err error, msg string) error {
	if err == nil {
		return nil
	}
	return apperr.Wrap(apperr.Internal, msg, err)
}

func mapNotFound(err error, msg string) error {
	if err == nil {
		return nil
	}
	if err == pgx.ErrNoRows {
		return apperr.New(apperr.NotFound, msg)
	}
	return wrapDB(err, msg)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

var (
	_ storage.MetadataStore     = (*Store)(nil)
	_ storage.ArtifactMetaStore = (*Store)(nil)
	_ storage.DocumentStore     = (*Store)(nil)
	_ storage.TxRunner          = (*Store)(nil)
)