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
	OK         bool   `json:"ok"`
	Service    string `json:"service"`
	APIVersion string `json:"api_version,omitempty"`
	Time       string `json:"time,omitempty"`
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
	DenseBackend  string      `json:"dense_backend,omitempty"`
	SparseBackend string      `json:"sparse_backend,omitempty"`
	FocusID       string      `json:"focus_id,omitempty"`
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
	Run       json.RawMessage `json:"run"`
	PackID    string          `json:"pack_id,omitempty"`
	ModelText string          `json:"model_text,omitempty"`
	ToolCall  json.RawMessage `json:"tool_call,omitempty"`
	VerifyOK  bool            `json:"verify_ok"`
	MetaKind  string          `json:"meta_kind,omitempty"`
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
