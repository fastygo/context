package jobs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/agentruntime/jobs"
)

func TestStartCompleteAndCancel(t *testing.T) {
	dir := t.TempDir()
	started := make(chan struct{})
	block := make(chan struct{})

	reg, err := jobs.Open(dir, func(ctx context.Context, spec jobs.Spec) (jobs.Outcome, error) {
		close(started)
		select {
		case <-block:
			return jobs.Outcome{RunID: "run_ok", PackID: "pack_ok", ModelText: "done", VerifyOK: true}, nil
		case <-ctx.Done():
			return jobs.Outcome{}, ctx.Err()
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	j, err := reg.Start(jobs.Spec{ProjectID: "p1", Query: "q", Owner: "owner-a"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("runner not started")
	}
	close(block)
	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err := reg.Get(j.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == agentruntime.RunStatusCompleted {
			if got.RunID != "run_ok" || got.Owner != "owner-a" {
				t.Fatalf("%#v", got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("not completed: %#v", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(dir, jobs.DirRel, string(j.ID)+".json")); err != nil {
		t.Fatal(err)
	}

	started2 := make(chan struct{})
	reg2, err := jobs.Open(t.TempDir(), func(ctx context.Context, spec jobs.Spec) (jobs.Outcome, error) {
		close(started2)
		<-ctx.Done()
		return jobs.Outcome{}, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	j2, err := reg2.Start(jobs.Spec{ProjectID: "p1", Query: "q", Owner: "o"})
	if err != nil {
		t.Fatal(err)
	}
	<-started2
	if _, err := reg2.Cancel(j2.ID); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for {
		got, err := reg2.Get(j2.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == agentruntime.RunStatusCancelled {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("want cancelled: %#v", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestOwnerRequired(t *testing.T) {
	reg, err := jobs.Open(t.TempDir(), func(ctx context.Context, spec jobs.Spec) (jobs.Outcome, error) {
		return jobs.Outcome{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.Start(jobs.Spec{ProjectID: "p1", Query: "q"})
	if err == nil {
		t.Fatal("want owner required")
	}
}
