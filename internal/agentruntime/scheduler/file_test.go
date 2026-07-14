package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/fastygo/context/internal/agentruntime/scheduler"
	"github.com/fastygo/context/internal/ids"
)

func TestFileStoreSurvivesRestartAndTick(t *testing.T) {
	dir := t.TempDir()
	store, err := scheduler.OpenFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Unix(1000, 0).UTC()
	next := now.Add(-time.Second)
	spec := scheduler.Spec{
		ID: "sched_1", ProjectID: "p1", Owner: "ops", Query: "ping",
		Kind: scheduler.KindOnceAt, Enabled: true, NextRunAt: &next,
	}
	if err := store.Put(ctx, spec); err != nil {
		t.Fatal(err)
	}
	// Simulate process restart with a new store handle on same dir.
	store2, err := scheduler.OpenFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	var started []ids.ScheduleID
	fired, err := scheduler.Tick(ctx, store2, func(ctx context.Context, s scheduler.Spec) (ids.JobID, error) {
		started = append(started, s.ID)
		return "job_from_sched", nil
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(fired) != 1 || fired[0] != "sched_1" || len(started) != 1 {
		t.Fatalf("fired=%v started=%v", fired, started)
	}
	got, err := store2.Get(ctx, "sched_1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled || got.LastJobID != "job_from_sched" {
		t.Fatalf("once_at must disable after fire: %#v", got)
	}
}

func TestIntervalAdvancesNextRun(t *testing.T) {
	dir := t.TempDir()
	store, err := scheduler.OpenFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Unix(2000, 0).UTC()
	next := now
	spec := scheduler.Spec{
		ID: "sched_i", ProjectID: "p1", Owner: "ops", Query: "tick",
		Kind: scheduler.KindInterval, Enabled: true, IntervalSeconds: 60, NextRunAt: &next,
	}
	if err := store.Put(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if _, err := scheduler.Tick(ctx, store, func(ctx context.Context, s scheduler.Spec) (ids.JobID, error) {
		return "job_i", nil
	}, now); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "sched_i")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.NextRunAt == nil || !got.NextRunAt.Equal(now.Add(60*time.Second)) {
		t.Fatalf("%#v", got)
	}
}

func TestFireEvent(t *testing.T) {
	dir := t.TempDir()
	store, err := scheduler.OpenFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	spec := scheduler.Spec{
		ID: "sched_e", ProjectID: "p1", Owner: "ops", Query: "on ingest",
		Kind: scheduler.KindEvent, Enabled: true, EventType: "ingest.completed",
	}
	if err := store.Put(ctx, spec); err != nil {
		t.Fatal(err)
	}
	fired, err := scheduler.FireEvent(ctx, store, func(ctx context.Context, s scheduler.Spec) (ids.JobID, error) {
		return "job_e", nil
	}, "p1", "ingest.completed", time.Unix(3000, 0).UTC())
	if err != nil || len(fired) != 1 {
		t.Fatalf("fired=%v err=%v", fired, err)
	}
}
