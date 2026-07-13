-- Chunk 11: durable metadata schema (ADR-0002, ADR-0022, ADR-0023).
-- Dense vectors remain in context_dense_vectors (Chunk 10).
-- Linguistic/lexicographic rows store adapter-neutral JSON payloads.

CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS projects (
  id text PRIMARY KEY,
  name text NOT NULL,
  active_snapshot_id text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sources (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_id text NOT NULL,
  source_type text NOT NULL,
  path_key text NOT NULL,
  uri text NOT NULL DEFAULT '',
  trust_level text NOT NULL,
  media_type text NOT NULL DEFAULT '',
  checksum text NOT NULL DEFAULT '',
  temporal_start timestamptz,
  temporal_end timestamptz,
  temporal_basis text,
  ingested_at timestamptz,
  PRIMARY KEY (project_id, source_id)
);
CREATE INDEX IF NOT EXISTS sources_project_idx ON sources(project_id);
CREATE INDEX IF NOT EXISTS sources_path_key_idx ON sources(project_id, path_key);
CREATE INDEX IF NOT EXISTS sources_temporal_idx ON sources(project_id, temporal_basis, temporal_start, temporal_end);

CREATE TABLE IF NOT EXISTS artifacts_meta (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  artifact_id text NOT NULL,
  source_id text NOT NULL DEFAULT '',
  media_type text NOT NULL,
  byte_size bigint NOT NULL DEFAULT 0,
  checksum text NOT NULL,
  storage_uri text NOT NULL,
  artifact_type text NOT NULL DEFAULT 'blob',
  schema_id text NOT NULL DEFAULT '',
  PRIMARY KEY (project_id, artifact_id)
);
CREATE INDEX IF NOT EXISTS artifacts_meta_schema_idx ON artifacts_meta(project_id, artifact_type, schema_id);
CREATE INDEX IF NOT EXISTS artifacts_meta_source_idx ON artifacts_meta(project_id, source_id);

CREATE TABLE IF NOT EXISTS artifact_lineage (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  output_artifact_id text NOT NULL,
  input_artifact_ids jsonb NOT NULL DEFAULT '[]',
  source_refs jsonb NOT NULL DEFAULT '[]',
  context_pack_id text NOT NULL DEFAULT '',
  agent_run_id text NOT NULL DEFAULT '',
  tool_call_id text NOT NULL DEFAULT '',
  generator_id text NOT NULL,
  generator_version text NOT NULL,
  transformation_kind text NOT NULL,
  created_at timestamptz NOT NULL,
  PRIMARY KEY (project_id, output_artifact_id),
  FOREIGN KEY (project_id, output_artifact_id)
    REFERENCES artifacts_meta(project_id, artifact_id)
);
CREATE INDEX IF NOT EXISTS artifact_lineage_project_idx ON artifact_lineage(project_id);
CREATE INDEX IF NOT EXISTS artifact_lineage_created_idx ON artifact_lineage(project_id, created_at);

CREATE TABLE IF NOT EXISTS chunks (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  chunk_id text NOT NULL,
  source_id text NOT NULL,
  artifact_id text NOT NULL,
  snapshot_id text NOT NULL,
  chunker_version text NOT NULL,
  span_start bigint NOT NULL,
  span_end bigint NOT NULL,
  text_checksum text NOT NULL,
  chunk_hash text NOT NULL,
  language text NOT NULL DEFAULT '',
  embedding_version text NOT NULL DEFAULT '',
  sparse_version text NOT NULL DEFAULT '',
  temporal_start timestamptz,
  temporal_end timestamptz,
  temporal_basis text,
  ingested_at timestamptz,
  PRIMARY KEY (project_id, chunk_id)
);
CREATE INDEX IF NOT EXISTS chunks_snapshot_idx ON chunks(project_id, snapshot_id);
CREATE INDEX IF NOT EXISTS chunks_source_idx ON chunks(project_id, source_id);
CREATE INDEX IF NOT EXISTS chunks_language_idx ON chunks(project_id, language);
CREATE INDEX IF NOT EXISTS chunks_temporal_idx ON chunks(project_id, temporal_basis, temporal_start, temporal_end);

CREATE TABLE IF NOT EXISTS index_snapshots (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  snapshot_id text NOT NULL,
  parent_snapshot_id text NOT NULL DEFAULT '',
  status text NOT NULL,
  source_merkle_root text NOT NULL DEFAULT '',
  chunk_set_hash text NOT NULL DEFAULT '',
  source_merkle_algo text NOT NULL DEFAULT '',
  chunk_set_merkle_algo text NOT NULL DEFAULT '',
  parser_version text NOT NULL DEFAULT '',
  chunker_version text NOT NULL DEFAULT '',
  embed_model_version text NOT NULL DEFAULT '',
  morph_version text NOT NULL DEFAULT '',
  sparse_index_ref jsonb NOT NULL DEFAULT '{}',
  vector_namespace jsonb NOT NULL DEFAULT '{}',
  dense_enabled boolean NOT NULL DEFAULT false,
  sparse_enabled boolean NOT NULL DEFAULT false,
  failure_reason text NOT NULL DEFAULT '',
  PRIMARY KEY (project_id, snapshot_id)
);
CREATE INDEX IF NOT EXISTS index_snapshots_status_idx ON index_snapshots(project_id, status);

CREATE TABLE IF NOT EXISTS manifest_nodes (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  snapshot_id text NOT NULL,
  path_key text NOT NULL,
  node_hash text NOT NULL,
  source_id text NOT NULL DEFAULT '',
  child_keys jsonb NOT NULL DEFAULT '[]',
  is_leaf boolean NOT NULL DEFAULT false,
  PRIMARY KEY (project_id, snapshot_id, path_key)
);
CREATE INDEX IF NOT EXISTS manifest_nodes_snapshot_idx ON manifest_nodes(project_id, snapshot_id);

CREATE TABLE IF NOT EXISTS chunk_aliases (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  snapshot_id text NOT NULL,
  alias text NOT NULL,
  chunk_id text NOT NULL,
  PRIMARY KEY (project_id, snapshot_id, alias)
);
CREATE INDEX IF NOT EXISTS chunk_aliases_chunk_idx ON chunk_aliases(project_id, snapshot_id, chunk_id);

CREATE TABLE IF NOT EXISTS context_packs (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  pack_id text NOT NULL,
  task_id text NOT NULL DEFAULT '',
  retrieval_plan_id text NOT NULL,
  purpose text NOT NULL DEFAULT '',
  checksum text NOT NULL DEFAULT '',
  payload jsonb NOT NULL,
  PRIMARY KEY (project_id, pack_id)
);

CREATE TABLE IF NOT EXISTS agent_runs (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  run_id text NOT NULL,
  task_id text NOT NULL DEFAULT '',
  mode text NOT NULL,
  status text NOT NULL,
  focus_id text NOT NULL DEFAULT '',
  policy_id text NOT NULL DEFAULT '',
  pack_id text NOT NULL DEFAULT '',
  parent_run_id text NOT NULL DEFAULT '',
  owner text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  error text NOT NULL DEFAULT '',
  PRIMARY KEY (project_id, run_id)
);
CREATE INDEX IF NOT EXISTS agent_runs_status_idx ON agent_runs(project_id, status, updated_at);

CREATE TABLE IF NOT EXISTS tool_calls (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  tool_call_id text NOT NULL,
  run_id text NOT NULL DEFAULT '',
  tool_name text NOT NULL,
  input_artifact_id text NOT NULL DEFAULT '',
  output_artifact_id text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT '',
  decision jsonb NOT NULL DEFAULT '{}',
  risk_level text NOT NULL DEFAULT '',
  error text NOT NULL DEFAULT '',
  PRIMARY KEY (project_id, tool_call_id)
);
CREATE INDEX IF NOT EXISTS tool_calls_run_idx ON tool_calls(project_id, run_id);

CREATE TABLE IF NOT EXISTS trace_events (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  run_id text NOT NULL,
  event_id text NOT NULL,
  event_type text NOT NULL,
  event_ts timestamptz NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}',
  analyzer_version text NOT NULL DEFAULT '',
  dictionary_version text NOT NULL DEFAULT '',
  feature_scheme text NOT NULL DEFAULT '',
  query_expansion_ver text NOT NULL DEFAULT '',
  sense_mapping_version text NOT NULL DEFAULT '',
  concept_mapping_ver text NOT NULL DEFAULT '',
  attestation_version text NOT NULL DEFAULT '',
  snapshot_id text NOT NULL DEFAULT '',
  PRIMARY KEY (project_id, run_id, event_id)
);
CREATE INDEX IF NOT EXISTS trace_events_run_ts_idx ON trace_events(project_id, run_id, event_ts);
CREATE INDEX IF NOT EXISTS trace_events_analyzer_idx ON trace_events(project_id, analyzer_version);
CREATE INDEX IF NOT EXISTS trace_events_dict_idx ON trace_events(project_id, dictionary_version);

CREATE TABLE IF NOT EXISTS evaluations (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  evaluation_id text NOT NULL,
  run_id text NOT NULL DEFAULT '',
  pack_id text NOT NULL DEFAULT '',
  kind text NOT NULL DEFAULT '',
  score double precision,
  payload jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, evaluation_id)
);
CREATE INDEX IF NOT EXISTS evaluations_run_idx ON evaluations(project_id, run_id);

-- Adapter-neutral linguistic / lexicographic documents (no language/dict imports).
CREATE TABLE IF NOT EXISTS meta_documents (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  kind text NOT NULL,
  document_id text NOT NULL,
  language text NOT NULL DEFAULT '',
  lexeme_id text NOT NULL DEFAULT '',
  sense_id text NOT NULL DEFAULT '',
  concept_id text NOT NULL DEFAULT '',
  region text NOT NULL DEFAULT '',
  register text NOT NULL DEFAULT '',
  time_period text NOT NULL DEFAULT '',
  lexicon_source_id text NOT NULL DEFAULT '',
  source_authority text NOT NULL DEFAULT '',
  analyzer_version text NOT NULL DEFAULT '',
  dictionary_version text NOT NULL DEFAULT '',
  snapshot_id text NOT NULL DEFAULT '',
  chunk_id text NOT NULL DEFAULT '',
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (project_id, kind, document_id)
);
CREATE INDEX IF NOT EXISTS meta_documents_kind_idx ON meta_documents(project_id, kind);
CREATE INDEX IF NOT EXISTS meta_documents_sense_idx ON meta_documents(project_id, sense_id);
CREATE INDEX IF NOT EXISTS meta_documents_concept_idx ON meta_documents(project_id, concept_id);
CREATE INDEX IF NOT EXISTS meta_documents_lexeme_idx ON meta_documents(project_id, lexeme_id);
CREATE INDEX IF NOT EXISTS meta_documents_lang_idx ON meta_documents(project_id, language);
CREATE INDEX IF NOT EXISTS meta_documents_region_idx ON meta_documents(project_id, region, register, time_period);
CREATE INDEX IF NOT EXISTS meta_documents_lexsrc_idx ON meta_documents(project_id, lexicon_source_id);
CREATE INDEX IF NOT EXISTS meta_documents_analyzer_idx ON meta_documents(project_id, analyzer_version, dictionary_version);
