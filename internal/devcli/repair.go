package devcli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

const (
	RepairModeRebuild     = "rebuild"
	RepairModeRetryFailed = "retry-failed"
	RepairTargetAll       = "all"
	RepairTargetDense     = "dense"
	RepairTargetSparse    = "sparse"
)

// RepairResult is CLI/HTTP JSON for index rebuild/repair (Chunk 23).
type RepairResult struct {
	OK               bool           `json:"ok"`
	Mode             string         `json:"mode"`
	Target           string         `json:"target"`
	ProjectID        ids.ProjectID  `json:"project_id"`
	SnapshotID       ids.SnapshotID `json:"snapshot_id"`
	ParentSnapshotID ids.SnapshotID `json:"parent_snapshot_id,omitempty"`
	Chunks           int            `json:"chunks"`
	DenseUpserted    bool           `json:"dense_upserted"`
	DenseSkipped     bool           `json:"dense_skipped"`
	DenseSkipReason  string         `json:"dense_skip_reason,omitempty"`
	SparseUpserted   bool           `json:"sparse_upserted"`
	SparseSkipped    bool           `json:"sparse_skipped"`
	SparseSkipReason string         `json:"sparse_skip_reason,omitempty"`
	Activated        bool           `json:"activated"`
	ClearedLastFailed bool          `json:"cleared_last_failed,omitempty"`
	Notes            string         `json:"notes,omitempty"`
}

// Repair rebuilds index payloads for the active ready snapshot, or retries a
// retained failed commit under a new snapshot_id (ADR-0021).
func Repair(dataDir, projectID, mode, target string) (RepairResult, error) {
	if mode == "" {
		mode = RepairModeRebuild
	}
	if target == "" {
		target = RepairTargetAll
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	target = strings.ToLower(strings.TrimSpace(target))
	switch mode {
	case RepairModeRebuild, RepairModeRetryFailed:
	default:
		return RepairResult{}, apperr.New(apperr.Validation, "mode must be rebuild|retry-failed")
	}
	switch target {
	case RepairTargetAll, RepairTargetDense, RepairTargetSparse:
	default:
		return RepairResult{}, apperr.New(apperr.Validation, "target must be all|dense|sparse")
	}

	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return RepairResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return RepairResult{}, apperr.New(apperr.Validation, "project id mismatch")
	}

	ctx := context.Background()
	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return RepairResult{}, err
	}

	switch mode {
	case RepairModeRebuild:
		return repairRebuild(ctx, ws, st, cfg, target)
	default:
		return repairRetryFailed(ctx, ws, st, cfg, target)
	}
}

func repairRebuild(ctx context.Context, ws Workspace, st State, cfg config.StorageConfig, target string) (RepairResult, error) {
	snapID := st.Project.ActiveSnapshotID
	if snapID == "" {
		snapID = st.Snapshot.ID
	}
	if snapID == "" {
		return RepairResult{}, apperr.New(apperr.Validation, "no active snapshot; run ingest")
	}
	if st.Snapshot.ID == snapID && st.Snapshot.Status == foundation.SnapshotFailed {
		return RepairResult{}, apperr.New(apperr.Conflict, "active pointer cannot be failed; use mode=retry-failed")
	}
	chunks := chunksForSnapshot(st.Chunks, snapID)
	if len(chunks) == 0 {
		return RepairResult{}, apperr.New(apperr.NotFound, "no chunks for active snapshot")
	}
	if err := beginIndexOp(ws, &st, RepairModeRebuild, target); err != nil {
		return RepairResult{}, err
	}
	defer func() { _ = clearIndexOp(ws, &st) }()

	res := RepairResult{
		OK:         true,
		Mode:       RepairModeRebuild,
		Target:     target,
		ProjectID:  st.Project.ID,
		SnapshotID: snapID,
		Chunks:     len(chunks),
		Notes:      "idempotent re-upsert; active ready snapshot remains searchable (C1)",
	}
	if err := applyIndexUpserts(ctx, cfg, st.Project.ID, chunks, target, &res); err != nil {
		return RepairResult{}, err
	}
	res.Activated = false // rebuild never flips active
	return res, nil
}

func repairRetryFailed(ctx context.Context, ws Workspace, st State, cfg config.StorageConfig, target string) (RepairResult, error) {
	if st.LastFailed == nil || len(st.LastFailed.Chunks) == 0 {
		return RepairResult{}, apperr.New(apperr.NotFound, "no last_failed attempt to retry; run ingest that fails index write or use mode=rebuild")
	}
	failed := st.LastFailed.Snapshot
	parent := st.Project.ActiveSnapshotID
	newID := ids.SnapshotID(fmt.Sprintf("snap_repair_%d", time.Now().UTC().UnixNano()))
	chunks := make([]IndexedChunk, 0, len(st.LastFailed.Chunks))
	for _, ch := range st.LastFailed.Chunks {
		ch.SnapshotID = newID
		chunks = append(chunks, ch)
	}
	building := failed
	building.ID = newID
	building.ProjectID = st.Project.ID
	building.ParentSnapshotID = parent
	building.Status = foundation.SnapshotBuilding
	building.FailureReason = ""
	building.VectorNamespace.SnapshotID = newID

	res := RepairResult{
		OK:               true,
		Mode:             RepairModeRetryFailed,
		Target:           target,
		ProjectID:        st.Project.ID,
		SnapshotID:       newID,
		ParentSnapshotID: parent,
		Chunks:           len(chunks),
		Notes:            "new snapshot_id from last_failed (ADR-0021); prior active remains searchable until activate",
	}
	if err := beginIndexOp(ws, &st, RepairModeRetryFailed, target); err != nil {
		return RepairResult{}, err
	}
	if err := applyIndexUpserts(ctx, cfg, st.Project.ID, chunks, target, &res); err != nil {
		// Keep LastFailed; do not activate. Clear rebuild marker.
		failedRetry := building
		failedRetry.Status = foundation.SnapshotFailed
		failedRetry.FailureReason = "repair_write_failed"
		st.LastFailed = &FailedAttempt{Snapshot: failedRetry, Chunks: chunks}
		st.IndexOp = nil
		_ = ws.Save(st)
		return RepairResult{}, apperr.Wrap(apperr.Internal, "repair index write failed", err)
	}

	ready := building
	ready.Status = foundation.SnapshotReady
	ready.FailureReason = ""
	// Preserve merkle roots from the failed attempt (same chunk set).
	st.Snapshot = ready
	st.Project.ActiveSnapshotID = newID
	st.Chunks = chunks
	st.LastFailed = nil
	st.IndexOp = nil
	if err := ws.Save(st); err != nil {
		return RepairResult{}, err
	}
	res.Activated = true
	res.ClearedLastFailed = true

	handle, err := OpenMetadata(ctx)
	if err == nil {
		defer handle.Close()
		if handle.UsesPostgres() {
			_ = handle.Store.PutProject(ctx, st.Project)
			_ = handle.Store.PutSnapshot(ctx, ready)
		}
	}
	return res, nil
}

func applyIndexUpserts(
	ctx context.Context,
	cfg config.StorageConfig,
	projectID ids.ProjectID,
	chunks []IndexedChunk,
	target string,
	res *RepairResult,
) error {
	wantDense := target == RepairTargetAll || target == RepairTargetDense
	wantSparse := target == RepairTargetAll || target == RepairTargetSparse

	if wantDense {
		if !denseEnabledByEnv() {
			res.DenseSkipped = true
			res.DenseSkipReason = "CONTEXT_ENABLE_DENSE not set"
		} else {
			if err := commitDenseVectors(ctx, cfg, projectID, chunks[0].SnapshotID, chunks); err != nil {
				return err
			}
			res.DenseUpserted = true
		}
	} else {
		res.DenseSkipped = true
		res.DenseSkipReason = "target excludes dense"
	}

	if wantSparse {
		sparseH, err := OpenSparse(ctx, nil)
		if err != nil {
			return err
		}
		defer sparseH.Closer()
		if !sparseH.UsesFTS {
			res.SparseSkipped = true
			res.SparseSkipReason = "CONTEXT_SPARSE_KIND is not postgres_fts"
		} else {
			if err := sparseH.UpsertChunks(ctx, projectID, chunks); err != nil {
				return err
			}
			res.SparseUpserted = true
		}
	} else {
		res.SparseSkipped = true
		res.SparseSkipReason = "target excludes sparse"
	}
	return nil
}

func chunksForSnapshot(chunks []IndexedChunk, snap ids.SnapshotID) []IndexedChunk {
	out := make([]IndexedChunk, 0, len(chunks))
	for _, ch := range chunks {
		if ch.SnapshotID == snap {
			out = append(out, ch)
		}
	}
	return out
}
