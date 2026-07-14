package devcli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/indexing/lifecycle"
)

func TestIndexStatusReadyAndDegraded(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")

	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.md"), []byte("# A\n\nhello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_idx", "Idx"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_idx", ""); err != nil {
		t.Fatal(err)
	}
	ready, err := devcli.IndexStatus(data, "proj_idx")
	if err != nil {
		t.Fatal(err)
	}
	if ready.Phase != lifecycle.PhaseReady || !ready.SearchAvailable {
		t.Fatalf("%#v", ready)
	}

	ws := devcli.Workspace{DataDir: data}
	st, err := ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	st.LastFailed = &devcli.FailedAttempt{
		Snapshot: st.Snapshot,
	}
	st.LastFailed.Snapshot.Status = foundation.SnapshotFailed
	st.LastFailed.Snapshot.FailureReason = "dense write failed"
	if err := ws.Save(st); err != nil {
		t.Fatal(err)
	}
	deg, err := devcli.IndexStatus(data, "proj_idx")
	if err != nil {
		t.Fatal(err)
	}
	if deg.Phase != lifecycle.PhaseDegraded || !deg.SearchAvailable {
		t.Fatalf("degraded must keep search: %#v", deg)
	}

	st, err = ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	st.IndexOp = &lifecycle.Op{Kind: "rebuild", Target: "all"}
	if err := ws.Save(st); err != nil {
		t.Fatal(err)
	}
	reb, err := devcli.IndexStatus(data, "proj_idx")
	if err != nil {
		t.Fatal(err)
	}
	if reb.Phase != lifecycle.PhaseRebuilding || !reb.SearchAvailable {
		t.Fatalf("rebuild must keep search: %#v", reb)
	}
}
