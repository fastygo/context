// Package scheduler defines durable schedule ports for background AgentRun
// (stabilization C8). Execution remains in-process via jobs.Registry; only the
// schedule definition and next-fire time survive process restart.
package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
)

// Kind selects when a schedule fires.
type Kind string

const (
	KindOnceAt   Kind = "once_at"
	KindInterval Kind = "interval"
	KindEvent    Kind = "event"
)

func (k Kind) Validate() error {
	switch k {
	case KindOnceAt, KindInterval, KindEvent:
		return nil
	case "":
		return fmt.Errorf("schedule kind: empty")
	default:
		return fmt.Errorf("schedule kind: unknown %q", k)
	}
}

// Spec is a durable trigger that enqueues background AgentRun jobs.
type Spec struct {
	ID              ids.ScheduleID `json:"id"`
	ProjectID       ids.ProjectID  `json:"project_id"`
	Owner           string         `json:"owner"`
	Query           string         `json:"query"`
	FocusID         string         `json:"focus_id,omitempty"`
	Kind            Kind           `json:"kind"`
	Enabled         bool           `json:"enabled"`
	NextRunAt       *time.Time     `json:"next_run_at,omitempty"`
	IntervalSeconds int            `json:"interval_seconds,omitempty"`
	EventType       string         `json:"event_type,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	LastFiredAt     *time.Time     `json:"last_fired_at,omitempty"`
	LastJobID       ids.JobID      `json:"last_job_id,omitempty"`
}

// Validate checks required fields for persistence.
func (s Spec) Validate() error {
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if err := s.ProjectID.Validate(); err != nil {
		return err
	}
	if s.Owner == "" {
		return apperr.New(apperr.Validation, "owner required")
	}
	if s.Query == "" {
		return apperr.New(apperr.Validation, "query required")
	}
	if err := s.Kind.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "kind", err)
	}
	switch s.Kind {
	case KindOnceAt:
		if s.NextRunAt == nil || s.NextRunAt.IsZero() {
			return apperr.New(apperr.Validation, "once_at requires next_run_at")
		}
	case KindInterval:
		if s.IntervalSeconds < 1 {
			return apperr.New(apperr.Validation, "interval requires interval_seconds >= 1")
		}
		if s.NextRunAt == nil || s.NextRunAt.IsZero() {
			return apperr.New(apperr.Validation, "interval requires next_run_at")
		}
	case KindEvent:
		if s.EventType == "" {
			return apperr.New(apperr.Validation, "event requires event_type")
		}
	}
	return nil
}

// Store is the replaceable schedule persistence port.
type Store interface {
	Put(ctx context.Context, spec Spec) error
	Get(ctx context.Context, id ids.ScheduleID) (Spec, error)
	List(ctx context.Context, projectID ids.ProjectID) ([]Spec, error)
	Delete(ctx context.Context, id ids.ScheduleID) error
	// Due returns enabled time-based schedules with NextRunAt <= now.
	Due(ctx context.Context, now time.Time) ([]Spec, error)
	// MarkFired records last fire and advances/disables the schedule.
	MarkFired(ctx context.Context, id ids.ScheduleID, at time.Time, jobID ids.JobID) error
}

// Enqueuer starts a background job for a due schedule (usually jobs.Registry.Start).
type Enqueuer func(ctx context.Context, spec Spec) (ids.JobID, error)
