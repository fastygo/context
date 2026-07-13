-- Chunk 24 / ADR-0025: optional tenant_id on projects (outer ACL/quota root).
ALTER TABLE projects
  ADD COLUMN IF NOT EXISTS tenant_id text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS projects_tenant_id_idx ON projects (tenant_id)
  WHERE tenant_id <> '';
