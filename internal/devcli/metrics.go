package devcli

import (
	"strconv"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/evals/golden"
	"github.com/fastygo/context/internal/ops"
)

// EvalHistoryResult is CLI/HTTP JSON for eval-history.
type EvalHistoryResult struct {
	Records []ops.EvalRecord `json:"records"`
	PathKey string           `json:"path_key,omitempty"`
}

// MetricsResult is CLI/HTTP JSON for metrics.
type MetricsResult = ops.Metrics

// EvalHistory lists newest-first eval records from a JSONL path.
func EvalHistory(historyPath string, limit int) (EvalHistoryResult, error) {
	if historyPath == "" {
		return EvalHistoryResult{}, apperr.New(apperr.Validation, "history path required")
	}
	recs, err := ops.ListEval(historyPath, limit)
	if err != nil {
		return EvalHistoryResult{}, err
	}
	return EvalHistoryResult{Records: recs, PathKey: ops.DefaultHistoryRel}, nil
}

// Metrics builds a workspace operational snapshot (no host paths).
func Metrics(dataDir string) (MetricsResult, error) {
	if dataDir == "" {
		return MetricsResult{}, apperr.New(apperr.Validation, "data dir required")
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return MetricsResult{}, err
	}
	histPath := ops.HistoryPath(dataDir)
	all, err := ops.ListEval(histPath, 0)
	if err != nil {
		return MetricsResult{}, err
	}
	m := MetricsResult{
		OK:                 true,
		ProjectID:          string(st.Project.ID),
		ProjectName:        st.Project.Name,
		ActiveSnapshotID:   string(st.Project.ActiveSnapshotID),
		SnapshotID:         string(st.Snapshot.ID),
		SnapshotStatus:     string(st.Snapshot.Status),
		Chunks:             len(st.Chunks),
		Packs:              len(st.Packs),
		Runs:               len(st.Runs),
		Focuses:            len(st.Focuses),
		Traces:             len(st.Traces),
		EvalHistoryCount:   len(all),
		EvalHistoryPathKey: ops.DefaultHistoryRel,
	}
	if len(all) > 0 {
		last := all[0]
		m.LastEval = &last
	}
	if st.LastFailed != nil {
		m.HasLastFailed = true
		m.LastFailedReason = st.LastFailed.Snapshot.FailureReason
	}
	return m, nil
}

func summaryFromReport(rep golden.Report, dur time.Duration) ops.EvalRecord {
	passed, failed := 0, 0
	for _, c := range rep.Cases {
		if c.Passed {
			passed++
		} else {
			failed++
		}
	}
	return ops.EvalRecord{
		RecordedAt: time.Now().UTC(),
		SuiteID:    rep.SuiteID,
		OK:         rep.OK,
		Passed:     passed,
		Failed:     failed,
		Total:      len(rep.Cases),
		DurationMS: dur.Milliseconds(),
		Summary:    rep.Summary,
	}
}

// ParseLimit parses a non-negative limit; empty -> defaultLimit.
func ParseLimit(s string, defaultLimit int) int {
	if s == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return defaultLimit
	}
	return n
}
