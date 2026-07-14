package devcli_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/agentruntime/scheduler"
	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/ids"
)

func TestScheduleTickEnqueuesJobAfterRestart(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")
	t.Setenv("CONTEXT_COMPLETER_KIND", "localecho")

	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.md"), []byte("# A\n\nZEBRA42 token\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_sched", "Sched"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_sched", ""); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	put, err := devcli.SchedulePut(data, scheduler.Spec{
		ID: "sched_restart", ProjectID: "proj_sched", Owner: "lab",
		Query: "ZEBRA42", Kind: scheduler.KindOnceAt, Enabled: true, NextRunAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if put.Schedule.ID != "sched_restart" {
		t.Fatalf("%#v", put)
	}
	tick, err := devcli.ScheduleTick(data)
	if err != nil {
		t.Fatal(err)
	}
	if tick.FiredN != 1 {
		t.Fatalf("tick=%#v", tick)
	}
	deadline := time.Now().Add(3 * time.Second)
	var jobID ids.JobID
	for {
		list, err := devcli.JobList(data, "proj_sched")
		if err != nil {
			t.Fatal(err)
		}
		if len(list.Jobs) > 0 {
			jobID = list.Jobs[0].ID
			if list.Jobs[0].Status == agentruntime.RunStatusCompleted || list.Jobs[0].Status == agentruntime.RunStatusFailed {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("job not finished: %#v", list.Jobs)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if jobID == "" {
		t.Fatal("expected job from schedule")
	}
}
