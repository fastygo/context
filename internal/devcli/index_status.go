package devcli

import (
	"time"

	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/lifecycle"
	"github.com/fastygo/context/internal/policy/isolation"
)

// IndexStatusResult is CLI/HTTP JSON for index lifecycle explain (C1).
type IndexStatusResult struct {
	lifecycle.Report
	OK bool `json:"ok"`
}

// IndexStatus reports phase (ready|degraded|rebuilding|failed|empty) and whether
// search remains available. Rebuild does not clear active ready snapshots.
func IndexStatus(dataDir, projectID string) (IndexStatusResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return IndexStatusResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return IndexStatusResult{}, err
	}
	failedReason := ""
	if st.LastFailed != nil {
		failedReason = st.LastFailed.Snapshot.FailureReason
	}
	liveChunks := 0
	snap := st.Project.ActiveSnapshotID
	if snap == "" {
		snap = st.Snapshot.ID
	}
	for _, ch := range st.Chunks {
		if ch.SnapshotID == snap {
			liveChunks++
		}
	}
	rep := lifecycle.Evaluate(lifecycle.Input{
		ProjectID:           st.Project.ID,
		ActiveSnapshotID:    st.Project.ActiveSnapshotID,
		SnapshotID:          st.Snapshot.ID,
		SnapshotStatus:      st.Snapshot.Status,
		DenseEnabled:        st.Snapshot.DenseEnabled,
		SparseEnabled:       st.Snapshot.SparseEnabled,
		ChunkCount:          liveChunks,
		TombstonedSourceIDs: len(st.TombstonedSourceIDs),
		HasLastFailed:       st.LastFailed != nil,
		LastFailedReason:    failedReason,
		IndexOp:             st.IndexOp,
	})
	return IndexStatusResult{OK: true, Report: rep}, nil
}

// beginIndexOp persists a rebuild marker; search stays on the active snapshot.
func beginIndexOp(ws Workspace, st *State, kind, target string) error {
	st.IndexOp = &lifecycle.Op{Kind: kind, Target: target, StartedAt: time.Now().UTC()}
	return ws.Save(*st)
}

func clearIndexOp(ws Workspace, st *State) error {
	st.IndexOp = nil
	return ws.Save(*st)
}
