// Package agentruntime defines AgentRun orchestration records.
package agentruntime

import (
	"fmt"
	"time"

	"github.com/fastygo/context/internal/ids"
)

// RunMode classifies foreground vs background execution.
type RunMode string

const (
	RunModeForeground RunMode = "foreground"
	RunModeBackground RunMode = "background"
	RunModeScheduled  RunMode = "scheduled"
	RunModeEvent      RunMode = "event"
)

// RunStatus is the agent run lifecycle state.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

func (s RunStatus) Validate() error {
	switch s {
	case RunStatusPending, RunStatusRunning, RunStatusCompleted, RunStatusFailed, RunStatusCancelled:
		return nil
	case "":
		return fmt.Errorf("run_status: empty")
	default:
		return fmt.Errorf("run_status: unknown %q", s)
	}
}

// AgentRun is a foreground, background, scheduled, or event-triggered execution trace.
type AgentRun struct {
	ID          ids.RunID     `json:"id"`
	ProjectID   ids.ProjectID `json:"project_id"`
	TaskID      ids.TaskID    `json:"task_id,omitempty"`
	Mode        RunMode       `json:"mode"`
	Status      RunStatus     `json:"status"`
	FocusID     ids.FocusID   `json:"focus_id,omitempty"`
	PolicyID    ids.PolicyID  `json:"policy_id,omitempty"`
	PackID      ids.PackID    `json:"pack_id,omitempty"`
	ParentRunID ids.RunID     `json:"parent_run_id,omitempty"`
	Owner       string        `json:"owner,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Error       string        `json:"error,omitempty"`
}

func (r AgentRun) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.ProjectID.Validate(); err != nil {
		return err
	}
	if r.Mode == "" {
		return fmt.Errorf("agent_run mode: empty")
	}
	return r.Status.Validate()
}
