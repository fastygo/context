-- Stabilization C1: soft-delete tombstones for sources (ADR-0028).
ALTER TABLE sources ADD COLUMN IF NOT EXISTS tombstoned_at timestamptz;
CREATE INDEX IF NOT EXISTS sources_live_idx ON sources(project_id) WHERE tombstoned_at IS NULL;
