package devcli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
)

func TestRepairRebuildOffline(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	data := setupRepairWorkspace(t)
	res, err := devcli.Repair(data, "proj_repair", devcli.RepairModeRebuild, devcli.RepairTargetAll)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Chunks < 1 {
		t.Fatalf("%#v", res)
	}
	if !res.DenseSkipped || !res.SparseSkipped {
		t.Fatalf("expected skips offline: %#v", res)
	}
	if res.Activated {
		t.Fatal("rebuild must not flip active")
	}
}

func TestRepairRetryFailedNewSnapshot(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	data := setupRepairWorkspace(t)
	ws := devcli.Workspace{DataDir: data}
	st, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	prevActive := st.Project.ActiveSnapshotID
	failedChunks := make([]devcli.IndexedChunk, len(st.Chunks))
	copy(failedChunks, st.Chunks)
	for i := range failedChunks {
		failedChunks[i].SnapshotID = "snap_failed_1"
		failedChunks[i].Text = failedChunks[i].Text + " repaired"
	}
	st.LastFailed = &devcli.FailedAttempt{
		Snapshot: indexing.IndexSnapshot{
			ID:               "snap_failed_1",
			ProjectID:        st.Project.ID,
			Status:           foundation.SnapshotFailed,
			FailureReason:    "dense_write_failed",
			ParserVersion:    "p1",
			ChunkerVersion:   "c1",
			SourceMerkleRoot: st.Snapshot.SourceMerkleRoot,
			ChunkSetHash:     st.Snapshot.ChunkSetHash,
			SourceMerkleAlgo: foundation.SourceMerkleAlgo,
			ChunkSetMerkleAlgo: foundation.ChunkSetMerkleAlgo,
		},
		Chunks: failedChunks,
	}
	if err := ws.Save(st); err != nil {
		t.Fatal(err)
	}

	res, err := devcli.Repair(data, "proj_repair", devcli.RepairModeRetryFailed, "all")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || !res.Activated || !res.ClearedLastFailed {
		t.Fatalf("%#v", res)
	}
	if res.SnapshotID == "" || res.SnapshotID == "snap_failed_1" || res.SnapshotID == prevActive {
		t.Fatalf("expected new snapshot id, got %#v", res)
	}
	st2, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	if st2.LastFailed != nil {
		t.Fatal("last_failed should be cleared")
	}
	if st2.Project.ActiveSnapshotID != res.SnapshotID {
		t.Fatalf("active=%s want %s", st2.Project.ActiveSnapshotID, res.SnapshotID)
	}
	if st2.Snapshot.Status != foundation.SnapshotReady {
		t.Fatalf("status=%s", st2.Snapshot.Status)
	}
	if len(st2.Chunks) == 0 || st2.Chunks[0].SnapshotID != res.SnapshotID {
		t.Fatalf("chunks not remapped: %#v", st2.Chunks)
	}
}

func TestRepairRetryFailedMissing(t *testing.T) {
	data := setupRepairWorkspace(t)
	_, err := devcli.Repair(data, "proj_repair", devcli.RepairModeRetryFailed, "all")
	if err == nil {
		t.Fatal("expected error")
	}
}

func setupRepairWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "notes.md"), []byte("# Repair\n\ntoken REPAIR42 here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_repair", "Repair"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_repair", ""); err != nil {
		t.Fatal(err)
	}
	_ = ids.ProjectID("proj_repair")
	return data
}
