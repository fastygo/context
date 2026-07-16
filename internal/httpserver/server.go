// Package httpserver exposes a thin HTTP+JSON boundary over proven CLI contracts
// (ADR-0024). Handlers call internal/devcli; clients must not import other
// internal packages.
package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/fastygo/context/internal/agentruntime/scheduler"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/policy/isolation"
	"github.com/fastygo/context/internal/retrieval"
)

// Config binds the process-local workspace and optional shared secret.
type Config struct {
	// DataDir is the workspace root (--data), owned by the server process.
	DataDir string
	// Token, when non-empty, requires Bearer or X-Context-Token on /v1/*.
	Token string
	// EvalOut is optional path for POST /v1/eval report write (may be empty).
	EvalOut string
}

// Server is the HTTP application surface.
type Server struct {
	cfg Config
	mux *http.ServeMux
}

// New returns a Server with routes registered.
func New(cfg Config) (*Server, error) {
	if strings.TrimSpace(cfg.DataDir) == "" {
		return nil, apperr.New(apperr.Validation, "data dir required")
	}
	abs, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return nil, apperr.Wrap(apperr.Validation, "data dir", err)
	}
	cfg.DataDir = abs
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	// Best-effort: on serve start, enqueue overdue schedules (C8 durability model).
	go func() { _, _ = devcli.ScheduleTick(cfg.DataDir) }()
	return s, nil
}

// Handler returns the root HTTP handler (auth + routes + API version header).
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(APIVersionHeader, APIVersion)
		if r.URL.Path == "/health" {
			s.handleHealth(w, r)
			return
		}
		if s.cfg.Token != "" && strings.HasPrefix(r.URL.Path, "/v1/") {
			if !s.authorized(r) {
				writeErr(w, http.StatusUnauthorized, apperr.New(apperr.Permission, "unauthorized"))
				return
			}
		}
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) authorized(r *http.Request) bool {
	want := s.cfg.Token
	if ah := r.Header.Get("Authorization"); strings.HasPrefix(ah, "Bearer ") {
		if strings.TrimSpace(strings.TrimPrefix(ah, "Bearer ")) == want {
			return true
		}
	}
	return r.Header.Get("X-Context-Token") == want
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /v1/status", s.handleStatus)
	s.mux.HandleFunc("GET /v1/index", s.handleIndexStatus)
	s.mux.HandleFunc("GET /v1/ready", s.handleReady)
	s.mux.HandleFunc("POST /v1/search", s.handleSearch)
	s.mux.HandleFunc("POST /v1/context-pack", s.handlePack)
	s.mux.HandleFunc("POST /v1/agent-run", s.handleAgent)
	s.mux.HandleFunc("GET /v1/trace", s.handleTrace)
	s.mux.HandleFunc("PUT /v1/focus", s.handleFocusPut)
	s.mux.HandleFunc("GET /v1/focus", s.handleFocusGet)
	s.mux.HandleFunc("GET /v1/focuses", s.handleFocusList)
	s.mux.HandleFunc("POST /v1/eval", s.handleEval)
	s.mux.HandleFunc("GET /v1/eval/history", s.handleEvalHistory)
	s.mux.HandleFunc("GET /v1/metrics", s.handleMetrics)
	s.mux.HandleFunc("GET /v1/quota", s.handleQuota)
	s.mux.HandleFunc("POST /v1/repair", s.handleRepair)
	s.mux.HandleFunc("POST /v1/inspect", s.handleInspect)
	s.mux.HandleFunc("POST /v1/ingest", s.handleIngest)
	s.mux.HandleFunc("POST /v1/sources/tombstone", s.handleTombstoneSource)
	s.mux.HandleFunc("POST /v1/snapshot/export", s.handleSnapshotExport)
	s.mux.HandleFunc("POST /v1/snapshot/import", s.handleSnapshotImport)
	s.mux.HandleFunc("POST /v1/project/export", s.handleProjectExport)
	s.mux.HandleFunc("POST /v1/project/delete", s.handleProjectDelete)
	s.mux.HandleFunc("POST /v1/jobs", s.handleJobStart)
	s.mux.HandleFunc("GET /v1/jobs", s.handleJobList)
	s.mux.HandleFunc("GET /v1/jobs/{id}", s.handleJobGet)
	s.mux.HandleFunc("POST /v1/jobs/{id}/cancel", s.handleJobCancel)
	s.mux.HandleFunc("PUT /v1/schedules", s.handleSchedulePut)
	s.mux.HandleFunc("GET /v1/schedules", s.handleScheduleList)
	s.mux.HandleFunc("DELETE /v1/schedules/{id}", s.handleScheduleDelete)
	s.mux.HandleFunc("POST /v1/schedules/tick", s.handleScheduleTick)
	s.mux.HandleFunc("POST /v1/schedules/fire", s.handleScheduleFire)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	body := map[string]any{
		"ok":          true,
		"service":     "context-serve",
		"api_version": APIVersion,
		"time":        time.Now().UTC().Format(time.RFC3339),
	}
	if rep, err := devcli.Ready(r.Context()); err == nil {
		body["ready"] = rep.Ready
		body["degraded"] = rep.Degraded
		body["backends"] = rep.Backends
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	projectQ := r.URL.Query().Get("project_id")
	if projectQ != "" {
		st, err := (devcli.Workspace{DataDir: s.cfg.DataDir}).Load()
		if err != nil {
			writeAppErr(w, err)
			return
		}
		if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectQ)); err != nil {
			writeAppErr(w, err)
			return
		}
	}
	rep, err := devcli.Ready(r.Context())
	if err != nil {
		writeAppErr(w, err)
		return
	}
	status := http.StatusOK
	if !rep.Ready {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, rep)
}

// StatusResponse is workspace ingest status without host filesystem paths.
type StatusResponse struct {
	OK               bool           `json:"ok"`
	ProjectID        ids.ProjectID  `json:"project_id"`
	ProjectName      string         `json:"project_name,omitempty"`
	ActiveSnapshotID ids.SnapshotID `json:"active_snapshot_id,omitempty"`
	SnapshotID       ids.SnapshotID `json:"snapshot_id,omitempty"`
	SnapshotStatus   string         `json:"snapshot_status,omitempty"`
	Chunks           int            `json:"chunks"`
	Packs            int            `json:"packs"`
	Runs             int            `json:"runs"`
	Focuses          int            `json:"focuses"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	st, err := (devcli.Workspace{DataDir: s.cfg.DataDir}).Load()
	if err != nil {
		writeAppErr(w, err)
		return
	}
	projectQ := r.URL.Query().Get("project_id")
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectQ)); err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, StatusResponse{
		OK:               true,
		ProjectID:        st.Project.ID,
		ProjectName:      st.Project.Name,
		ActiveSnapshotID: st.Project.ActiveSnapshotID,
		SnapshotID:       st.Snapshot.ID,
		SnapshotStatus:   string(st.Snapshot.Status),
		Chunks:           len(st.Chunks),
		Packs:            len(st.Packs),
		Runs:             len(st.Runs),
		Focuses:          len(st.Focuses),
	})
}

func (s *Server) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	res, err := devcli.IndexStatus(s.cfg.DataDir, r.URL.Query().Get("project_id"))
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type searchRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	Mode      string `json:"mode,omitempty"`
	FocusID   string `json:"focus_id,omitempty"`
	// Lang selects the expansion language adapter (additive v1, ADR-0043).
	// mode=query also honors an in-query lang: directive, which wins.
	Lang string `json:"lang,omitempty"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Query == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "query required"))
		return
	}
	res, err := devcli.SearchWithLang(s.cfg.DataDir, req.ProjectID, req.Query, req.Mode, req.FocusID, req.Lang)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type packRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	FocusID   string `json:"focus_id,omitempty"`
}

func (s *Server) handlePack(w http.ResponseWriter, r *http.Request) {
	var req packRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Query == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "query required"))
		return
	}
	res, err := devcli.BuildPack(s.cfg.DataDir, req.ProjectID, req.Query, req.FocusID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	var req packRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Query == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "query required"))
		return
	}
	res, err := devcli.AgentRun(s.cfg.DataDir, req.ProjectID, req.Query, req.FocusID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleTrace(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "run_id required"))
		return
	}
	res, err := devcli.Trace(s.cfg.DataDir, projectID, runID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type focusPutRequest struct {
	ProjectID string                 `json:"project_id"`
	Focus     retrieval.FocusProfile `json:"focus"`
}

func (s *Server) handleFocusPut(w http.ResponseWriter, r *http.Request) {
	var req focusPutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Focus.Objective == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "focus.objective required"))
		return
	}
	if req.Focus.RequiredTrustLevel == "" {
		req.Focus.RequiredTrustLevel = foundation.TrustProject
	}
	if req.Focus.ContextBudget.MaxItems == 0 {
		req.Focus.ContextBudget = retrieval.Budget{MaxItems: 8, MaxChars: 4000}
	}
	res, err := devcli.PutFocus(s.cfg.DataDir, req.ProjectID, req.Focus)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleFocusGet(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	focusID := r.URL.Query().Get("focus_id")
	if focusID == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "focus_id required"))
		return
	}
	focus, kind, err := devcli.GetFocus(s.cfg.DataDir, projectID, focusID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"focus": focus, "meta_kind": kind})
}

func (s *Server) handleFocusList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	res, err := devcli.ListFocus(s.cfg.DataDir, projectID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleEval(w http.ResponseWriter, r *http.Request) {
	out := s.cfg.EvalOut
	var body struct {
		Out string `json:"out,omitempty"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		_ = decodeJSON(r, &body)
		if body.Out != "" {
			writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "client out path not allowed; configure server eval_out"))
			return
		}
	}
	history := filepath.Join(s.cfg.DataDir, "ops", "eval_history.jsonl")
	res, err := devcli.RunEval(out, history)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	status := http.StatusOK
	if !res.Report.OK {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, res)
}

func (s *Server) handleEvalHistory(w http.ResponseWriter, r *http.Request) {
	limit := devcli.ParseLimit(r.URL.Query().Get("limit"), 20)
	history := filepath.Join(s.cfg.DataDir, "ops", "eval_history.jsonl")
	res, err := devcli.EvalHistory(history, limit)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	projectQ := r.URL.Query().Get("project_id")
	res, err := devcli.Metrics(s.cfg.DataDir)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	if err := isolation.RequireProjectMatch(ids.ProjectID(res.ProjectID), ids.ProjectID(projectQ)); err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleQuota(w http.ResponseWriter, r *http.Request) {
	projectQ := r.URL.Query().Get("project_id")
	res, err := devcli.QuotaStatus(s.cfg.DataDir)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	m, err := devcli.Metrics(s.cfg.DataDir)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	if err := isolation.RequireProjectMatch(ids.ProjectID(m.ProjectID), ids.ProjectID(projectQ)); err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type repairRequest struct {
	ProjectID string `json:"project_id"`
	Mode      string `json:"mode,omitempty"`
	Target    string `json:"target,omitempty"`
}

func (s *Server) handleRepair(w http.ResponseWriter, r *http.Request) {
	var req repairRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := devcli.Repair(s.cfg.DataDir, req.ProjectID, req.Mode, req.Target)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type inspectRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query,omitempty"`
	FocusID   string `json:"focus_id,omitempty"`
	PackID    string `json:"pack_id,omitempty"`
}

func (s *Server) handleInspect(w http.ResponseWriter, r *http.Request) {
	var req inspectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := devcli.Inspect(s.cfg.DataDir, req.ProjectID, req.Query, req.FocusID, req.PackID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type jobStartRequest struct {
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	FocusID   string `json:"focus_id,omitempty"`
	Owner     string `json:"owner"`
}

func (s *Server) handleJobStart(w http.ResponseWriter, r *http.Request) {
	var req jobStartRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.Query == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "query required"))
		return
	}
	res, err := devcli.JobStart(s.cfg.DataDir, req.ProjectID, req.Query, req.FocusID, req.Owner)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, res)
}

func (s *Server) handleJobList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	res, err := devcli.JobList(s.cfg.DataDir, projectID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleJobGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	projectID := r.URL.Query().Get("project_id")
	res, err := devcli.JobStatus(s.cfg.DataDir, projectID, id)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleJobCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		var body struct {
			ProjectID string `json:"project_id"`
		}
		_ = decodeJSON(r, &body)
		projectID = body.ProjectID
	}
	res, err := devcli.JobCancel(s.cfg.DataDir, projectID, id)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleSchedulePut(w http.ResponseWriter, r *http.Request) {
	var spec scheduler.Spec
	if err := decodeJSON(r, &spec); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := devcli.SchedulePut(s.cfg.DataDir, spec)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleScheduleList(w http.ResponseWriter, r *http.Request) {
	res, err := devcli.ScheduleList(s.cfg.DataDir, r.URL.Query().Get("project_id"))
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleScheduleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	projectID := r.URL.Query().Get("project_id")
	if err := devcli.ScheduleDelete(s.cfg.DataDir, projectID, id); err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "schedule_id": id})
}

func (s *Server) handleScheduleTick(w http.ResponseWriter, r *http.Request) {
	res, err := devcli.ScheduleTick(s.cfg.DataDir)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleScheduleFire(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		EventType string `json:"event_type"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := devcli.ScheduleFireEvent(s.cfg.DataDir, req.ProjectID, req.EventType)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type ingestRequest struct {
	ProjectID string `json:"project_id"`
	// PathKey is relative to the workspace corpus root (ADR-0013). Empty = full corpus.
	PathKey string `json:"path_key,omitempty"`
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ws := devcli.Workspace{DataDir: s.cfg.DataDir}
	st, err := ws.Load()
	if err != nil {
		writeAppErr(w, err)
		return
	}
	ingestPath := ""
	if req.PathKey != "" {
		p, err := resolveCorpusPath(st.CorpusRoot, req.PathKey)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		ingestPath = p
	}
	st, err = devcli.Ingest(s.cfg.DataDir, req.ProjectID, ingestPath)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"project_id":  st.Project.ID,
		"snapshot_id": st.Snapshot.ID,
		"chunks":      len(st.Chunks),
		"status":      st.Snapshot.Status,
	})
}

type tombstoneRequest struct {
	ProjectID string `json:"project_id"`
	SourceID  string `json:"source_id"`
}

func (s *Server) handleTombstoneSource(w http.ResponseWriter, r *http.Request) {
	var req tombstoneRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := devcli.TombstoneSource(s.cfg.DataDir, req.ProjectID, req.SourceID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type snapshotExportRequest struct {
	ProjectID string `json:"project_id"`
}

func (s *Server) handleSnapshotExport(w http.ResponseWriter, r *http.Request) {
	var req snapshotExportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	meta, bundle, err := devcli.ExportSnapshotBundle(s.cfg.DataDir, req.ProjectID, "")
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              meta.OK,
		"project_id":      meta.ProjectID,
		"snapshot_id":     meta.SnapshotID,
		"chunks":          meta.Chunks,
		"bundle_checksum": meta.BundleChecksum,
		"bundle":          bundle,
	})
}

type snapshotImportRequest struct {
	ProjectID string          `json:"project_id"`
	Activate  bool            `json:"activate"`
	Bundle    json.RawMessage `json:"bundle"`
}

func (s *Server) handleSnapshotImport(w http.ResponseWriter, r *http.Request) {
	var req snapshotImportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Bundle) == 0 {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "bundle required"))
		return
	}
	res, err := devcli.ImportSnapshotBundleBytes(s.cfg.DataDir, req.ProjectID, req.Bundle, req.Activate)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type projectExportRequest struct {
	ProjectID string `json:"project_id"`
	OutPath   string `json:"out_path"`
}

func (s *Server) handleProjectExport(w http.ResponseWriter, r *http.Request) {
	var req projectExportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.OutPath == "" {
		writeErr(w, http.StatusBadRequest, apperr.New(apperr.Validation, "out_path required"))
		return
	}
	res, err := devcli.ExportProject(s.cfg.DataDir, req.ProjectID, req.OutPath)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type projectDeleteRequest struct {
	ProjectID        string `json:"project_id"`
	ConfirmProjectID string `json:"confirm_project_id"`
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	var req projectDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res, err := devcli.DeleteProject(s.cfg.DataDir, req.ProjectID, req.ConfirmProjectID)
	if err != nil {
		writeAppErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// resolveCorpusPath joins path_key under corpus root and rejects escapes / abs paths.
func resolveCorpusPath(corpusRoot, pathKey string) (string, error) {
	key := filepath.ToSlash(strings.TrimSpace(pathKey))
	if key == "" || key == "." {
		return "", apperr.New(apperr.Validation, "empty path_key")
	}
	if filepath.IsAbs(pathKey) || strings.HasPrefix(key, "/") || strings.Contains(key, ":") {
		return "", apperr.New(apperr.Validation, "absolute path_key not allowed")
	}
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", apperr.New(apperr.Validation, "path_key escapes corpus root")
	}
	absRoot, err := filepath.Abs(corpusRoot)
	if err != nil {
		return "", apperr.Wrap(apperr.Validation, "corpus root", err)
	}
	full := filepath.Join(absRoot, clean)
	rel, err := filepath.Rel(absRoot, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", apperr.New(apperr.Validation, "path_key escapes corpus root")
	}
	return full, nil
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperr.Wrap(apperr.Validation, "decode json", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeAppErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case apperr.Is(err, apperr.NotFound):
		status = http.StatusNotFound
	case apperr.Is(err, apperr.Validation):
		status = http.StatusBadRequest
	case apperr.Is(err, apperr.Permission):
		status = http.StatusForbidden
	case apperr.Is(err, apperr.Conflict):
		status = http.StatusConflict
	case apperr.Is(err, apperr.Unavailable), apperr.Is(err, apperr.Degraded):
		status = http.StatusServiceUnavailable
	}
	writeErr(w, status, err)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	code := apperr.Internal
	msg := err.Error()
	var ae *apperr.Error
	if errors.As(err, &ae) && ae != nil {
		code = ae.Code
		msg = ae.Message
		if ae.Err != nil && ae.Message != "" {
			msg = ae.Error()
		}
	}
	writeJSON(w, status, map[string]any{
		"ok":      false,
		"error":   code,
		"message": msg,
	})
}
