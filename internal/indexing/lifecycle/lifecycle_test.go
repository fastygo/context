package lifecycle_test

import (
	"testing"
	"time"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/lifecycle"
)

func TestEvaluateReady(t *testing.T) {
	t.Parallel()
	r := lifecycle.Evaluate(lifecycle.Input{
		ProjectID: "p1", ActiveSnapshotID: "s1", SnapshotID: "s1",
		SnapshotStatus: foundation.SnapshotReady, ChunkCount: 3,
	})
	if r.Phase != lifecycle.PhaseReady || !r.SearchAvailable {
		t.Fatalf("%#v", r)
	}
}

func TestEvaluateRebuildKeepsSearch(t *testing.T) {
	t.Parallel()
	r := lifecycle.Evaluate(lifecycle.Input{
		ProjectID: "p1", ActiveSnapshotID: "s1", SnapshotID: "s1",
		SnapshotStatus: foundation.SnapshotReady, ChunkCount: 2,
		IndexOp: &lifecycle.Op{Kind: "rebuild", Target: "all", StartedAt: time.Now().UTC()},
	})
	if r.Phase != lifecycle.PhaseRebuilding || !r.SearchAvailable {
		t.Fatalf("rebuild must keep search: %#v", r)
	}
}

func TestEvaluateDegradedLastFailed(t *testing.T) {
	t.Parallel()
	r := lifecycle.Evaluate(lifecycle.Input{
		ProjectID: "p1", ActiveSnapshotID: "s1", SnapshotID: "s1",
		SnapshotStatus: foundation.SnapshotReady, ChunkCount: 1,
		HasLastFailed: true, LastFailedReason: "sparse failed",
	})
	if r.Phase != lifecycle.PhaseDegraded || !r.SearchAvailable {
		t.Fatalf("%#v", r)
	}
}

func TestEvaluateEmpty(t *testing.T) {
	t.Parallel()
	r := lifecycle.Evaluate(lifecycle.Input{ProjectID: "p1"})
	if r.Phase != lifecycle.PhaseEmpty || r.SearchAvailable {
		t.Fatalf("%#v", r)
	}
}
