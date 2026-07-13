// Package jobs provides in-process background AgentRun jobs with cancel
// (Chunk 31). No cron, queue, or multi-process workers.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
)

const DirRel = "ops/jobs"

// Spec starts one background agent job.
type Spec struct {
	ProjectID ids.ProjectID `json:"project_id"`
	Query     string        `json:"query"`
	FocusID   string        `json:"focus_id,omitempty"`
	Owner     string        `json:"owner"`
}

// Outcome is returned by the injected runner when a job finishes.
type Outcome struct {
	RunID     ids.RunID  `json:"run_id,omitempty"`
	PackID    ids.PackID `json:"pack_id,omitempty"`
	ModelText string     `json:"model_text,omitempty"`
	VerifyOK  bool       `json:"verify_ok,omitempty"`
}

// Runner executes the agent path under a cancellable context.
type Runner func(ctx context.Context, spec Spec) (Outcome, error)

// Job is the Lab-facing background job record (no host paths).
type Job struct {
	ID        ids.JobID               `json:"id"`
	ProjectID ids.ProjectID           `json:"project_id"`
	Query     string                  `json:"query"`
	FocusID   string                  `json:"focus_id,omitempty"`
	Owner     string                  `json:"owner"`
	Status    agentruntime.RunStatus  `json:"status"`
	RunID     ids.RunID               `json:"run_id,omitempty"`
	PackID    ids.PackID              `json:"pack_id,omitempty"`
	ModelText string                  `json:"model_text,omitempty"`
	VerifyOK  bool                    `json:"verify_ok,omitempty"`
	Error     string                  `json:"error,omitempty"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type entry struct {
	job    Job
	cancel context.CancelFunc
}

// Registry is a process-local job map persisted under dataDir/ops/jobs.
type Registry struct {
	mu      sync.Mutex
	dataDir string
	runner  Runner
	entries map[ids.JobID]*entry
}

// Open loads existing job JSON from disk and attaches a runner for new starts.
func Open(dataDir string, runner Runner) (*Registry, error) {
	if dataDir == "" {
		return nil, apperr.New(apperr.Validation, "data dir required")
	}
	if runner == nil {
		return nil, apperr.New(apperr.Validation, "job runner required")
	}
	r := &Registry{
		dataDir: dataDir,
		runner:  runner,
		entries: make(map[ids.JobID]*entry),
	}
	dir := filepath.Join(dataDir, DirRel)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, apperr.Wrap(apperr.Internal, "list jobs", err)
	}
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, apperr.Wrap(apperr.Internal, "read job", err)
		}
		var j Job
		if err := json.Unmarshal(raw, &j); err != nil {
			return nil, apperr.Wrap(apperr.Internal, "decode job", err)
		}
		// Process restart: running/pending jobs are no longer live.
		if j.Status == agentruntime.RunStatusRunning || j.Status == agentruntime.RunStatusPending {
			j.Status = agentruntime.RunStatusFailed
			j.Error = "process restarted; in-process job lost"
			j.UpdatedAt = time.Now().UTC()
			_ = r.writeJob(j)
		}
		r.entries[j.ID] = &entry{job: j}
	}
	return r, nil
}

// Start validates owner/query, persists pending job, and runs AgentRun in a goroutine.
func (r *Registry) Start(spec Spec) (Job, error) {
	if err := spec.ProjectID.Validate(); err != nil {
		return Job{}, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if spec.Query == "" {
		return Job{}, apperr.New(apperr.Validation, "query required")
	}
	if spec.Owner == "" {
		return Job{}, apperr.New(apperr.Validation, "owner required for background jobs")
	}
	now := time.Now().UTC()
	id := ids.JobID(fmt.Sprintf("job_%d", now.UnixNano()))
	j := Job{
		ID:        id,
		ProjectID: spec.ProjectID,
		Query:     spec.Query,
		FocusID:   spec.FocusID,
		Owner:     spec.Owner,
		Status:    agentruntime.RunStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.entries[id] = &entry{job: j, cancel: cancel}
	r.mu.Unlock()
	if err := r.persist(id); err != nil {
		cancel()
		return Job{}, err
	}
	go r.execute(ctx, id, spec)
	return j, nil
}

func (r *Registry) execute(ctx context.Context, id ids.JobID, spec Spec) {
	r.setStatus(id, agentruntime.RunStatusRunning, "", Outcome{})
	out, err := r.runner(ctx, spec)
	if err != nil {
		status := agentruntime.RunStatusFailed
		if ctx.Err() != nil {
			status = agentruntime.RunStatusCancelled
		}
		r.setStatus(id, status, err.Error(), Outcome{})
		return
	}
	r.setStatus(id, agentruntime.RunStatusCompleted, "", out)
}

func (r *Registry) setStatus(id ids.JobID, status agentruntime.RunStatus, errMsg string, out Outcome) {
	r.mu.Lock()
	ent, ok := r.entries[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	ent.job.Status = status
	ent.job.Error = errMsg
	ent.job.UpdatedAt = time.Now().UTC()
	if out.RunID != "" {
		ent.job.RunID = out.RunID
	}
	if out.PackID != "" {
		ent.job.PackID = out.PackID
	}
	if out.ModelText != "" {
		ent.job.ModelText = out.ModelText
	}
	ent.job.VerifyOK = out.VerifyOK
	j := ent.job
	r.mu.Unlock()
	_ = r.writeJob(j)
}

// Get returns a job by id.
func (r *Registry) Get(id ids.JobID) (Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ent, ok := r.entries[id]
	if !ok {
		return Job{}, apperr.New(apperr.NotFound, "job not found")
	}
	return ent.job, nil
}

// List returns newest-first jobs.
func (r *Registry) List() []Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Job, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.job)
	}
	// newest first
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Cancel requests cancellation of a pending/running job.
func (r *Registry) Cancel(id ids.JobID) (Job, error) {
	r.mu.Lock()
	ent, ok := r.entries[id]
	if !ok {
		r.mu.Unlock()
		return Job{}, apperr.New(apperr.NotFound, "job not found")
	}
	st := ent.job.Status
	cancel := ent.cancel
	r.mu.Unlock()
	if st == agentruntime.RunStatusCompleted || st == agentruntime.RunStatusFailed || st == agentruntime.RunStatusCancelled {
		return r.Get(id)
	}
	if cancel != nil {
		cancel()
	}
	// If still pending (runner not started), mark cancelled immediately.
	r.mu.Lock()
	ent, ok = r.entries[id]
	if ok && ent.job.Status == agentruntime.RunStatusPending {
		ent.job.Status = agentruntime.RunStatusCancelled
		ent.job.Error = "cancelled before start"
		ent.job.UpdatedAt = time.Now().UTC()
		j := ent.job
		r.mu.Unlock()
		_ = r.writeJob(j)
		return j, nil
	}
	r.mu.Unlock()
	return r.Get(id)
}

func (r *Registry) persist(id ids.JobID) error {
	r.mu.Lock()
	ent, ok := r.entries[id]
	if !ok {
		r.mu.Unlock()
		return apperr.New(apperr.NotFound, "job not found")
	}
	j := ent.job
	r.mu.Unlock()
	return r.writeJob(j)
}

func (r *Registry) writeJob(j Job) error {
	dir := filepath.Join(r.dataDir, DirRel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.Internal, "job dir", err)
	}
	raw, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "encode job", err)
	}
	path := filepath.Join(dir, string(j.ID)+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.Internal, "write job", err)
	}
	return nil
}
