-- Chunk 17: FocusProfile persistence
CREATE TABLE IF NOT EXISTS focus_profiles (
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  focus_id text NOT NULL,
  task_id text NOT NULL DEFAULT '',
  objective text NOT NULL DEFAULT '',
  payload jsonb NOT NULL,
  PRIMARY KEY (project_id, focus_id)
);
CREATE INDEX IF NOT EXISTS focus_profiles_task_idx ON focus_profiles(project_id, task_id);
