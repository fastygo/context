package devcli_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/devcli"
)

func TestJobStartCompletes(t *testing.T) {
	t.Setenv("CONTEXT_COMPLETER_KIND", "localecho")
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_FAIL_VECTOR", "")

	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.md"), []byte("# Alpha\n\nZEBRA42 token\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_job", "Jobs"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_job", ""); err != nil {
		t.Fatal(err)
	}

	start, err := devcli.JobStart(data, "proj_job", "ZEBRA42", "", "lab-owner")
	if err != nil {
		t.Fatal(err)
	}
	if start.Job.Owner != "lab-owner" {
		t.Fatalf("%#v", start.Job)
	}

	deadline := time.Now().Add(5 * time.Second)
	var got devcli.Job
	for {
		got, err = devcli.JobStatus(data, "proj_job", string(start.Job.ID))
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == agentruntime.RunStatusCompleted {
			break
		}
		if got.Status == agentruntime.RunStatusFailed {
			t.Fatalf("failed: %#v", got)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout: %#v", got)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got.PackID == "" || got.RunID == "" {
		t.Fatalf("%#v", got)
	}
	if got.RunID != "" {
		// Mode should be background on persisted run via agent path.
	}
	list, err := devcli.JobList(data, "proj_job")
	if err != nil || len(list.Jobs) < 1 {
		t.Fatalf("%#v %v", list, err)
	}
}

func TestJobStartRequiresOwner(t *testing.T) {
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	_ = os.MkdirAll(corpus, 0o755)
	_ = os.WriteFile(filepath.Join(corpus, "a.md"), []byte("x\n"), 0o644)
	if _, err := devcli.InitProject(data, corpus, "proj_job2", "Jobs"); err != nil {
		t.Fatal(err)
	}
	_, err := devcli.JobStart(data, "proj_job2", "x", "", "")
	if err == nil {
		t.Fatal("want owner required")
	}
}
