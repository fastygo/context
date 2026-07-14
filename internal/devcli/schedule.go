package devcli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fastygo/context/internal/agentruntime/jobs"
	"github.com/fastygo/context/internal/agentruntime/scheduler"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/policy/isolation"
)

var (
	schedMu    sync.Mutex
	schedStores = map[string]*scheduler.FileStore{}
)

// Schedule is the Lab-facing durable schedule record.
type Schedule = scheduler.Spec

func scheduleStore(dataDir string) (*scheduler.FileStore, error) {
	schedMu.Lock()
	defer schedMu.Unlock()
	if s, ok := schedStores[dataDir]; ok {
		return s, nil
	}
	s, err := scheduler.OpenFileStore(dataDir)
	if err != nil {
		return nil, err
	}
	schedStores[dataDir] = s
	return s, nil
}

func scheduleEnqueuer(dataDir string) scheduler.Enqueuer {
	return func(ctx context.Context, spec scheduler.Spec) (ids.JobID, error) {
		reg, err := jobRegistry(dataDir)
		if err != nil {
			return "", err
		}
		j, err := reg.Start(jobs.Spec{
			ProjectID: spec.ProjectID,
			Query:     spec.Query,
			FocusID:   spec.FocusID,
			Owner:     spec.Owner,
		})
		if err != nil {
			return "", err
		}
		return j.ID, nil
	}
}

// SchedulePutResult is CLI/HTTP JSON after upserting a schedule.
type SchedulePutResult struct {
	Schedule Schedule `json:"schedule"`
}

// ScheduleListResult lists durable schedules.
type ScheduleListResult struct {
	Schedules []Schedule `json:"schedules"`
}

// ScheduleTickResult reports which schedules fired in a tick.
type ScheduleTickResult struct {
	Fired   []ids.ScheduleID `json:"fired"`
	FiredN  int              `json:"fired_n"`
	At      time.Time        `json:"at"`
}

// SchedulePut upserts a durable schedule (file adapter).
func SchedulePut(dataDir string, spec scheduler.Spec) (SchedulePutResult, error) {
	if dataDir == "" {
		return SchedulePutResult{}, apperr.New(apperr.Validation, "data dir required")
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return SchedulePutResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, spec.ProjectID); err != nil {
		return SchedulePutResult{}, err
	}
	if spec.ID == "" {
		spec.ID = ids.ScheduleID(fmt.Sprintf("sched_%d", time.Now().UTC().UnixNano()))
	}
	store, err := scheduleStore(dataDir)
	if err != nil {
		return SchedulePutResult{}, err
	}
	if err := store.Put(context.Background(), spec); err != nil {
		return SchedulePutResult{}, err
	}
	got, err := store.Get(context.Background(), spec.ID)
	if err != nil {
		return SchedulePutResult{}, err
	}
	return SchedulePutResult{Schedule: got}, nil
}

// ScheduleList lists schedules for the workspace project.
func ScheduleList(dataDir, projectID string) (ScheduleListResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ScheduleListResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return ScheduleListResult{}, err
	}
	store, err := scheduleStore(dataDir)
	if err != nil {
		return ScheduleListResult{}, err
	}
	list, err := store.List(context.Background(), st.Project.ID)
	if err != nil {
		return ScheduleListResult{}, err
	}
	return ScheduleListResult{Schedules: list}, nil
}

// ScheduleDelete removes a schedule.
func ScheduleDelete(dataDir, projectID, scheduleID string) error {
	store, err := scheduleStore(dataDir)
	if err != nil {
		return err
	}
	got, err := store.Get(context.Background(), ids.ScheduleID(scheduleID))
	if err != nil {
		return err
	}
	if err := isolation.RequireProjectMatch(got.ProjectID, ids.ProjectID(projectID)); err != nil {
		return err
	}
	return store.Delete(context.Background(), ids.ScheduleID(scheduleID))
}

// ScheduleTick fires due time-based schedules into the in-process job registry.
// Safe to call after process restart — overdue schedules enqueue new jobs.
func ScheduleTick(dataDir string) (ScheduleTickResult, error) {
	store, err := scheduleStore(dataDir)
	if err != nil {
		return ScheduleTickResult{}, err
	}
	now := time.Now().UTC()
	fired, err := scheduler.Tick(context.Background(), store, scheduleEnqueuer(dataDir), now)
	if err != nil {
		return ScheduleTickResult{}, err
	}
	return ScheduleTickResult{Fired: fired, FiredN: len(fired), At: now}, nil
}

// ScheduleFireEvent triggers event-kind schedules for the project.
func ScheduleFireEvent(dataDir, projectID, eventType string) (ScheduleTickResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ScheduleTickResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return ScheduleTickResult{}, err
	}
	store, err := scheduleStore(dataDir)
	if err != nil {
		return ScheduleTickResult{}, err
	}
	now := time.Now().UTC()
	fired, err := scheduler.FireEvent(context.Background(), store, scheduleEnqueuer(dataDir), st.Project.ID, eventType, now)
	if err != nil {
		return ScheduleTickResult{}, err
	}
	return ScheduleTickResult{Fired: fired, FiredN: len(fired), At: now}, nil
}
