package contextkit

import "encoding/json"

// APIError is the JSON error body returned by context-serve.
type APIError struct {
	OK      bool   `json:"ok"`
	Code    string `json:"error"`
	Message string `json:"message"`
}

func (e APIError) Error() string {
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}

// HealthResponse is GET /health.
type HealthResponse struct {
	OK         bool            `json:"ok"`
	Service    string          `json:"service"`
	APIVersion string          `json:"api_version,omitempty"`
	Time       string          `json:"time,omitempty"`
	Ready      bool            `json:"ready,omitempty"`
	Degraded   bool            `json:"degraded,omitempty"`
	Backends   json.RawMessage `json:"backends,omitempty"`
}

// ReadyResult is GET /v1/ready.
type ReadyResult struct {
	OK       bool            `json:"ok"`
	Ready    bool            `json:"ready"`
	Degraded bool            `json:"degraded"`
	Backends json.RawMessage `json:"backends"`
}

// StatusResponse is GET /v1/status (no host filesystem paths).
type StatusResponse struct {
	OK               bool   `json:"ok"`
	ProjectID        string `json:"project_id"`
	ProjectName      string `json:"project_name,omitempty"`
	ActiveSnapshotID string `json:"active_snapshot_id,omitempty"`
	SnapshotID       string `json:"snapshot_id,omitempty"`
	SnapshotStatus   string `json:"snapshot_status,omitempty"`
	Chunks           int    `json:"chunks"`
	Packs            int    `json:"packs"`
	Runs             int    `json:"runs"`
	Focuses          int    `json:"focuses"`
}

// SearchRequest is POST /v1/search.
type SearchRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	Mode      string `json:"mode,omitempty"`
	FocusID   string `json:"focus_id,omitempty"`
}

// Candidate is a retrieval hit in SearchResult.
type Candidate struct {
	ChunkID     string          `json:"chunk_id"`
	MergedScore float64         `json:"merged_score"`
	TrustLevel  string          `json:"trust_level,omitempty"`
	TextChecksum string         `json:"text_checksum,omitempty"`
	SourceRef   json.RawMessage `json:"source_ref,omitempty"`
	Contributions json.RawMessage `json:"contributions,omitempty"`
}

// SearchResult is POST /v1/search response.
type SearchResult struct {
	ProjectID     string      `json:"project_id"`
	SnapshotID    string      `json:"snapshot_id"`
	Query         string      `json:"query"`
	Mode          string      `json:"mode"`
	Candidates    []Candidate `json:"candidates"`
	DenseBackend    string      `json:"dense_backend,omitempty"`
	SparseBackend   string      `json:"sparse_backend,omitempty"`
	FocusID         string      `json:"focus_id,omitempty"`
	Degraded        bool        `json:"degraded,omitempty"`
	DegradedReasons []string    `json:"degraded_reasons,omitempty"`
}

// PackRequest is POST /v1/context-pack and POST /v1/agent-run.
type PackRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	FocusID   string `json:"focus_id,omitempty"`
}

// PackResult is POST /v1/context-pack response.
type PackResult struct {
	ContextPack json.RawMessage `json:"context_pack"`
	FocusID     string          `json:"focus_id,omitempty"`
}

// AgentRunResult is POST /v1/agent-run response.
type AgentRunResult struct {
	Run         json.RawMessage `json:"run"`
	PackID      string          `json:"pack_id,omitempty"`
	ModelText   string          `json:"model_text,omitempty"`
	ToolCall    json.RawMessage `json:"tool_call,omitempty"`
	VerifyOK    bool            `json:"verify_ok"`
	MetaKind    string          `json:"meta_kind,omitempty"`
	Redacted    bool            `json:"redacted,omitempty"`
	RedactCount int             `json:"redact_count,omitempty"`
}

// RunID extracts run.id from AgentRunResult.Run.
func (r AgentRunResult) RunID() (string, error) {
	var bare struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(r.Run, &bare); err != nil {
		return "", err
	}
	return bare.ID, nil
}

// TraceResult is GET /v1/trace response.
type TraceResult struct {
	Run      json.RawMessage   `json:"run"`
	Events   []json.RawMessage `json:"events"`
	MetaKind string            `json:"meta_kind,omitempty"`
}

// FocusProfile is the public FocusProfile JSON shape (subset-safe).
type FocusProfile struct {
	ID                 string          `json:"id,omitempty"`
	ProjectID          string          `json:"project_id,omitempty"`
	TaskID             string          `json:"task_id,omitempty"`
	Objective          string          `json:"objective"`
	RequiredTrustLevel string          `json:"required_trust_level,omitempty"`
	CitationStrictness string          `json:"citation_strictness,omitempty"`
	ContextBudget      json.RawMessage `json:"context_budget,omitempty"`
}

// FocusPutRequest is PUT /v1/focus.
type FocusPutRequest struct {
	ProjectID string       `json:"project_id"`
	Focus     FocusProfile `json:"focus"`
}

// FocusPutResult is PUT /v1/focus response.
type FocusPutResult struct {
	Focus    FocusProfile `json:"focus"`
	MetaKind string       `json:"meta_kind,omitempty"`
}

// FocusGetResult is GET /v1/focus response.
type FocusGetResult struct {
	Focus    FocusProfile `json:"focus"`
	MetaKind string       `json:"meta_kind,omitempty"`
}

// FocusListResult is GET /v1/focuses response.
type FocusListResult struct {
	Focuses  []FocusProfile `json:"focuses"`
	MetaKind string         `json:"meta_kind,omitempty"`
}

// EvalResult is POST /v1/eval response.
type EvalResult struct {
	Report  json.RawMessage `json:"report"`
	Out     string          `json:"out,omitempty"`
	History string          `json:"history,omitempty"`
}

// EvalHistoryResult is GET /v1/eval/history response.
type EvalHistoryResult struct {
	Records []json.RawMessage `json:"records"`
	PathKey string            `json:"path_key,omitempty"`
}

// MetricsResult is GET /v1/metrics response.
type MetricsResult struct {
	OK                 bool            `json:"ok"`
	ProjectID          string          `json:"project_id,omitempty"`
	ProjectName        string          `json:"project_name,omitempty"`
	ActiveSnapshotID   string          `json:"active_snapshot_id,omitempty"`
	SnapshotID         string          `json:"snapshot_id,omitempty"`
	SnapshotStatus     string          `json:"snapshot_status,omitempty"`
	Chunks             int             `json:"chunks"`
	Packs              int             `json:"packs"`
	Runs               int             `json:"runs"`
	Focuses            int             `json:"focuses"`
	Traces             int             `json:"traces"`
	EvalHistoryCount   int             `json:"eval_history_count"`
	LastEval           json.RawMessage `json:"last_eval,omitempty"`
	EvalHistoryPathKey string          `json:"eval_history_path_key,omitempty"`
	HasLastFailed      bool            `json:"has_last_failed,omitempty"`
	LastFailedReason   string          `json:"last_failed_reason,omitempty"`
	Quota              json.RawMessage `json:"quota,omitempty"`
	Readiness          json.RawMessage `json:"readiness,omitempty"`
}

// QuotaResult is GET /v1/quota response.
type QuotaResult struct {
	OK       bool            `json:"ok"`
	Decision string          `json:"decision"`
	Limits   json.RawMessage `json:"limits"`
	Usage    json.RawMessage `json:"usage"`
	Breaches json.RawMessage `json:"breaches,omitempty"`
	Notes    []string        `json:"notes,omitempty"`
}

// RepairRequest is POST /v1/repair.
type RepairRequest struct {
	ProjectID string `json:"project_id"`
	Mode      string `json:"mode,omitempty"`   // rebuild | retry-failed
	Target    string `json:"target,omitempty"` // all | dense | sparse
}

// RepairResult is POST /v1/repair response.
type RepairResult struct {
	OK                bool   `json:"ok"`
	Mode              string `json:"mode"`
	Target            string `json:"target"`
	ProjectID         string `json:"project_id"`
	SnapshotID        string `json:"snapshot_id"`
	ParentSnapshotID  string `json:"parent_snapshot_id,omitempty"`
	Chunks            int    `json:"chunks"`
	DenseUpserted     bool   `json:"dense_upserted"`
	DenseSkipped      bool   `json:"dense_skipped"`
	DenseSkipReason   string `json:"dense_skip_reason,omitempty"`
	SparseUpserted    bool   `json:"sparse_upserted"`
	SparseSkipped     bool   `json:"sparse_skipped"`
	SparseSkipReason  string `json:"sparse_skip_reason,omitempty"`
	Activated         bool   `json:"activated"`
	ClearedLastFailed bool   `json:"cleared_last_failed,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

// InspectRequest is POST /v1/inspect.
type InspectRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query,omitempty"`
	FocusID   string `json:"focus_id,omitempty"`
	PackID    string `json:"pack_id,omitempty"`
}

// InspectResult is POST /v1/inspect (Lab-facing; nested details as raw JSON).
type InspectResult struct {
	OK           bool              `json:"ok"`
	ProjectID    string            `json:"project_id"`
	SnapshotID   string            `json:"snapshot_id,omitempty"`
	Query        string            `json:"query,omitempty"`
	Mode         string            `json:"mode"`
	FocusID      string            `json:"focus_id,omitempty"`
	PackID       string            `json:"pack_id,omitempty"`
	Purpose      string            `json:"purpose,omitempty"`
	PlanID       string            `json:"retrieval_plan_id,omitempty"`
	Budget       json.RawMessage   `json:"budget"`
	Instructions []string          `json:"instructions,omitempty"`
	Selected     []json.RawMessage `json:"selected"`
	Rejected     []json.RawMessage `json:"rejected,omitempty"`
	Candidates   []json.RawMessage `json:"candidates,omitempty"`
	Checksum     string            `json:"pack_checksum,omitempty"`
	Notes        []string          `json:"notes,omitempty"`
	Redacted     bool              `json:"redacted,omitempty"`
}

// IngestRequest is POST /v1/ingest (path_key relative to corpus only).
type IngestRequest struct {
	ProjectID string `json:"project_id"`
	PathKey   string `json:"path_key,omitempty"`
}

// IngestResult is POST /v1/ingest response.
type IngestResult struct {
	OK         bool   `json:"ok"`
	ProjectID  string `json:"project_id"`
	SnapshotID string `json:"snapshot_id"`
	Chunks     int    `json:"chunks"`
	Status     string `json:"status,omitempty"`
}

// TombstoneSourceRequest is POST /v1/sources/tombstone.
type TombstoneSourceRequest struct {
	ProjectID string `json:"project_id"`
	SourceID  string `json:"source_id"`
}

// TombstoneSourceResult is POST /v1/sources/tombstone response.
type TombstoneSourceResult struct {
	ProjectID    string `json:"project_id"`
	SourceID     string `json:"source_id"`
	Tombstoned   bool   `json:"tombstoned"`
	TombstonedAt string `json:"tombstoned_at"`
}

// SnapshotExportRequest is POST /v1/snapshot/export.
type SnapshotExportRequest struct {
	ProjectID string `json:"project_id"`
}

// SnapshotExportResult is POST /v1/snapshot/export response.
type SnapshotExportResult struct {
	OK             bool            `json:"ok"`
	ProjectID      string          `json:"project_id"`
	SnapshotID     string          `json:"snapshot_id"`
	Chunks         int             `json:"chunks"`
	BundleChecksum string          `json:"bundle_checksum"`
	Bundle         json.RawMessage `json:"bundle"`
}

// SnapshotImportRequest is POST /v1/snapshot/import.
type SnapshotImportRequest struct {
	ProjectID string          `json:"project_id"`
	Activate  bool            `json:"activate"`
	Bundle    json.RawMessage `json:"bundle"`
}

// SnapshotImportResult is POST /v1/snapshot/import response.
type SnapshotImportResult struct {
	OK         bool   `json:"ok"`
	ProjectID  string `json:"project_id"`
	SnapshotID string `json:"snapshot_id"`
	Chunks     int    `json:"chunks"`
	Activated  bool   `json:"activated"`
	Verified   bool   `json:"verified"`
}

// ProjectExportRequest is POST /v1/project/export.
type ProjectExportRequest struct {
	ProjectID string `json:"project_id"`
	OutPath   string `json:"out_path"`
}

// ProjectExportResult is POST /v1/project/export response.
type ProjectExportResult struct {
	OK         bool   `json:"ok"`
	ProjectID  string `json:"project_id"`
	SnapshotID string `json:"snapshot_id"`
	Chunks     int    `json:"chunks"`
	Focuses    int    `json:"focuses"`
	OutPath    string `json:"out_path,omitempty"`
}

// ProjectDeleteRequest is POST /v1/project/delete.
type ProjectDeleteRequest struct {
	ProjectID        string `json:"project_id"`
	ConfirmProjectID string `json:"confirm_project_id"`
}

// ProjectDeleteResult is POST /v1/project/delete response.
type ProjectDeleteResult struct {
	OK                bool   `json:"ok"`
	ProjectID         string `json:"project_id"`
	WorkspaceCleared  bool   `json:"workspace_cleared"`
	MetadataDeleted   bool   `json:"metadata_deleted"`
	ArtifactsDeleted  bool   `json:"artifacts_deleted"`
	SourcesTombstoned int    `json:"sources_tombstoned_before_delete"`
}

// SchedulePutRequest is PUT /v1/schedules (body is the schedule spec).
type SchedulePutRequest struct {
	ID              string  `json:"id,omitempty"`
	ProjectID       string  `json:"project_id"`
	Owner           string  `json:"owner"`
	Query           string  `json:"query"`
	FocusID         string  `json:"focus_id,omitempty"`
	Kind            string  `json:"kind"`
	Enabled         bool    `json:"enabled"`
	NextRunAt       *string `json:"next_run_at,omitempty"`
	IntervalSeconds int     `json:"interval_seconds,omitempty"`
	EventType       string  `json:"event_type,omitempty"`
}

// Schedule is a durable schedule record.
type Schedule struct {
	ID              string  `json:"id"`
	ProjectID       string  `json:"project_id"`
	Owner           string  `json:"owner"`
	Query           string  `json:"query"`
	FocusID         string  `json:"focus_id,omitempty"`
	Kind            string  `json:"kind"`
	Enabled         bool    `json:"enabled"`
	NextRunAt       *string `json:"next_run_at,omitempty"`
	IntervalSeconds int     `json:"interval_seconds,omitempty"`
	EventType       string  `json:"event_type,omitempty"`
	LastJobID       string  `json:"last_job_id,omitempty"`
}

// SchedulePutResult is PUT /v1/schedules response.
type SchedulePutResult struct {
	Schedule Schedule `json:"schedule"`
}

// ScheduleListResult is GET /v1/schedules response.
type ScheduleListResult struct {
	Schedules []Schedule `json:"schedules"`
}

// ScheduleTickResult is POST /v1/schedules/tick|fire response.
type ScheduleTickResult struct {
	Fired  []string `json:"fired"`
	FiredN int      `json:"fired_n"`
	At     string   `json:"at"`
}

// ScheduleFireRequest is POST /v1/schedules/fire.
type ScheduleFireRequest struct {
	ProjectID string `json:"project_id"`
	EventType string `json:"event_type"`
}

// JobStartRequest is POST /v1/jobs.
type JobStartRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	FocusID   string `json:"focus_id,omitempty"`
	Owner     string `json:"owner"`
}

// Job is a background AgentRun job record.
type Job struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	FocusID   string `json:"focus_id,omitempty"`
	Owner     string `json:"owner"`
	Status    string `json:"status"`
	RunID     string `json:"run_id,omitempty"`
	PackID    string `json:"pack_id,omitempty"`
	ModelText string `json:"model_text,omitempty"`
	VerifyOK  bool   `json:"verify_ok,omitempty"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// JobStartResult is POST /v1/jobs response.
type JobStartResult struct {
	Job Job `json:"job"`
}

// JobListResult is GET /v1/jobs response.
type JobListResult struct {
	Jobs []Job `json:"jobs"`
}
