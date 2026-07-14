// Package lifecycle reports project index health without expanding SnapshotStatus
// (ADR-0021). Stabilization C1: explain ready / degraded / rebuilding / failed.
package lifecycle

import (
	"time"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
)

// Phase is the Lab-facing index lifecycle summary (orthogonal to SnapshotStatus).
type Phase string

const (
	PhaseEmpty      Phase = "empty"
	PhaseReady      Phase = "ready"
	PhaseDegraded   Phase = "degraded"
	PhaseRebuilding Phase = "rebuilding"
	PhaseFailed     Phase = "failed"
)

// Reason codes are stable explain tokens for consumers.
const (
	ReasonNoSnapshot         = "no_snapshot"
	ReasonSnapshotNotReady   = "snapshot_not_ready"
	ReasonLastFailedRetained = "last_failed_retained"
	ReasonTombstonesPresent  = "tombstoned_sources_present"
	ReasonRebuildInProgress  = "rebuild_in_progress"
	ReasonSearchAvailable    = "search_available"
	ReasonSearchUnavailable  = "search_unavailable"
)

// Op is an in-flight index maintenance marker persisted on workspace state.
type Op struct {
	Kind      string    `json:"kind"` // rebuild | retry-failed
	Target    string    `json:"target,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

// Input is the minimal workspace view needed to compute a Report.
type Input struct {
	ProjectID           ids.ProjectID
	ActiveSnapshotID    ids.SnapshotID
	SnapshotID          ids.SnapshotID
	SnapshotStatus      foundation.SnapshotStatus
	DenseEnabled        bool
	SparseEnabled       bool
	ChunkCount          int
	TombstonedSourceIDs int
	HasLastFailed       bool
	LastFailedReason    string
	IndexOp             *Op
}

// Report is the explainable index lifecycle snapshot (no host paths).
type Report struct {
	ProjectID            ids.ProjectID             `json:"project_id"`
	ActiveSnapshotID     ids.SnapshotID            `json:"active_snapshot_id,omitempty"`
	SnapshotID           ids.SnapshotID            `json:"snapshot_id,omitempty"`
	SnapshotStatus       foundation.SnapshotStatus `json:"snapshot_status,omitempty"`
	Phase                Phase                     `json:"phase"`
	SearchAvailable      bool                      `json:"search_available"`
	Reasons              []string                  `json:"reasons"`
	DenseEnabled         bool                      `json:"dense_enabled"`
	SparseEnabled        bool                      `json:"sparse_enabled"`
	ChunkCount           int                       `json:"chunk_count"`
	TombstonedSourceCount int                      `json:"tombstoned_source_count"`
	HasLastFailed        bool                      `json:"has_last_failed"`
	LastFailedReason     string                    `json:"last_failed_reason,omitempty"`
	RebuildInProgress    bool                      `json:"rebuild_in_progress"`
	RebuildKind          string                    `json:"rebuild_kind,omitempty"`
	RebuildTarget        string                    `json:"rebuild_target,omitempty"`
}

// Evaluate derives phase and reasons. Rebuild never clears SearchAvailable when
// an active ready snapshot remains pointed (ADR-0021 rebuild semantics).
func Evaluate(in Input) Report {
	r := Report{
		ProjectID:             in.ProjectID,
		ActiveSnapshotID:      in.ActiveSnapshotID,
		SnapshotID:            in.SnapshotID,
		SnapshotStatus:        in.SnapshotStatus,
		DenseEnabled:          in.DenseEnabled,
		SparseEnabled:         in.SparseEnabled,
		ChunkCount:            in.ChunkCount,
		TombstonedSourceCount: in.TombstonedSourceIDs,
		HasLastFailed:         in.HasLastFailed,
		LastFailedReason:      in.LastFailedReason,
	}
	if in.IndexOp != nil {
		r.RebuildInProgress = true
		r.RebuildKind = in.IndexOp.Kind
		r.RebuildTarget = in.IndexOp.Target
		r.Reasons = append(r.Reasons, ReasonRebuildInProgress)
	}

	active := in.ActiveSnapshotID
	if active == "" {
		active = in.SnapshotID
	}
	if active == "" || in.ChunkCount == 0 {
		r.Phase = PhaseEmpty
		r.SearchAvailable = false
		r.Reasons = append(r.Reasons, ReasonNoSnapshot, ReasonSearchUnavailable)
		return r
	}

	r.SearchAvailable = in.SnapshotStatus.IsSearchableAsActive()
	if !r.SearchAvailable {
		r.Phase = PhaseFailed
		r.Reasons = append(r.Reasons, ReasonSnapshotNotReady, ReasonSearchUnavailable)
		return r
	}

	degraded := false
	if in.HasLastFailed {
		degraded = true
		r.Reasons = append(r.Reasons, ReasonLastFailedRetained)
	}
	if in.TombstonedSourceIDs > 0 {
		degraded = true
		r.Reasons = append(r.Reasons, ReasonTombstonesPresent)
	}

	if r.RebuildInProgress {
		r.Phase = PhaseRebuilding
		r.Reasons = append(r.Reasons, ReasonSearchAvailable)
		return r
	}
	if degraded {
		r.Phase = PhaseDegraded
		r.Reasons = append(r.Reasons, ReasonSearchAvailable)
		return r
	}
	r.Phase = PhaseReady
	r.Reasons = append(r.Reasons, ReasonSearchAvailable)
	return r
}
