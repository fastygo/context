// Package ops provides lightweight operational metrics and append-only eval
// history for Phase 3 regression tracking. History is JSONL under the process
// data dir (not a dashboard, not multi-tenant).
package ops

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ops/readiness"
	"github.com/fastygo/context/internal/policy/quota"
)

const (
	// DefaultHistoryRel is relative to a workspace data dir.
	DefaultHistoryRel = "ops/eval_history.jsonl"
)

// EvalRecord is one completed golden (or compatible) suite run summary.
type EvalRecord struct {
	RecordedAt time.Time `json:"recorded_at"`
	SuiteID    string    `json:"suite_id"`
	OK         bool      `json:"ok"`
	Passed     int       `json:"passed"`
	Failed     int       `json:"failed"`
	Total      int       `json:"total"`
	DurationMS int64     `json:"duration_ms"`
	Summary    string    `json:"summary,omitempty"`
}

// HistoryPath joins dataDir with the default relative history file.
func HistoryPath(dataDir string) string {
	return filepath.Join(dataDir, DefaultHistoryRel)
}

// AppendEval appends one record to a JSONL history file (create dirs as needed).
func AppendEval(path string, rec EvalRecord) error {
	if path == "" {
		return apperr.New(apperr.Validation, "eval history path required")
	}
	if rec.RecordedAt.IsZero() {
		rec.RecordedAt = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.Internal, "eval history dir", err)
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "encode eval record", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "open eval history", err)
	}
	defer f.Close()
	if _, err := f.Write(append(raw, '\n')); err != nil {
		return apperr.Wrap(apperr.Internal, "append eval history", err)
	}
	return nil
}

// ListEval returns the newest-first records, capped by limit (0 = all).
func ListEval(path string, limit int) ([]EvalRecord, error) {
	if path == "" {
		return nil, apperr.New(apperr.Validation, "eval history path required")
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.Internal, "read eval history", err)
	}
	defer f.Close()
	var all []EvalRecord
	sc := bufio.NewScanner(f)
	// Allow long JSON lines.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec EvalRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, apperr.Wrap(apperr.Internal, "decode eval history line", err)
		}
		all = append(all, rec)
	}
	if err := sc.Err(); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "scan eval history", err)
	}
	// Newest last on disk; reverse for newest-first.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// Metrics is a Lab-facing workspace health snapshot (no host paths).
type Metrics struct {
	OK                 bool        `json:"ok"`
	ProjectID          string      `json:"project_id,omitempty"`
	ProjectName        string      `json:"project_name,omitempty"`
	ActiveSnapshotID   string      `json:"active_snapshot_id,omitempty"`
	SnapshotID         string      `json:"snapshot_id,omitempty"`
	SnapshotStatus     string      `json:"snapshot_status,omitempty"`
	Chunks             int         `json:"chunks"`
	Packs              int         `json:"packs"`
	Runs               int         `json:"runs"`
	Focuses            int         `json:"focuses"`
	Traces             int         `json:"traces"`
	EvalHistoryCount   int         `json:"eval_history_count"`
	LastEval           *EvalRecord `json:"last_eval,omitempty"`
	EvalHistoryPathKey string      `json:"eval_history_path_key,omitempty"`
	HasLastFailed      bool             `json:"has_last_failed,omitempty"`
	LastFailedReason   string           `json:"last_failed_reason,omitempty"`
	Quota              *quota.Status    `json:"quota,omitempty"`
	Readiness          *readiness.Report `json:"readiness,omitempty"`
}
