package devcli

import (
	"context"
	"sync"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/agentruntime/jobs"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/policy/isolation"
)

var (
	jobRegMu sync.Mutex
	jobRegs  = map[string]*jobs.Registry{}
)

// Job is the Lab-facing background job record.
type Job = jobs.Job

// JobStartResult is CLI/HTTP JSON after starting a job.
type JobStartResult struct {
	Job Job `json:"job"`
}

// JobListResult is CLI/HTTP JSON for job list.
type JobListResult struct {
	Jobs []Job `json:"jobs"`
}

func jobRegistry(dataDir string) (*jobs.Registry, error) {
	jobRegMu.Lock()
	defer jobRegMu.Unlock()
	if r, ok := jobRegs[dataDir]; ok {
		return r, nil
	}
	r, err := jobs.Open(dataDir, func(ctx context.Context, spec jobs.Spec) (jobs.Outcome, error) {
		res, err := AgentRunContext(ctx, dataDir, string(spec.ProjectID), spec.Query, spec.FocusID, AgentRunOptions{
			Mode:  agentruntime.RunModeBackground,
			Owner: spec.Owner,
			TaskID: "bg-task",
		})
		if err != nil {
			return jobs.Outcome{}, err
		}
		return jobs.Outcome{
			RunID:     res.Run.ID,
			PackID:    res.PackID,
			ModelText: res.ModelText,
			VerifyOK:  res.VerifyOK,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	jobRegs[dataDir] = r
	return r, nil
}

// JobStart starts an in-process background AgentRun. Owner is required.
func JobStart(dataDir, projectID, query, focusID, owner string) (JobStartResult, error) {
	if dataDir == "" {
		return JobStartResult{}, apperr.New(apperr.Validation, "data dir required")
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return JobStartResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return JobStartResult{}, err
	}
	reg, err := jobRegistry(dataDir)
	if err != nil {
		return JobStartResult{}, err
	}
	j, err := reg.Start(jobs.Spec{
		ProjectID: st.Project.ID,
		Query:     query,
		FocusID:   focusID,
		Owner:     owner,
	})
	if err != nil {
		return JobStartResult{}, err
	}
	return JobStartResult{Job: j}, nil
}

// JobStatus returns one job by id.
func JobStatus(dataDir, projectID, jobID string) (Job, error) {
	reg, err := jobRegistry(dataDir)
	if err != nil {
		return Job{}, err
	}
	j, err := reg.Get(ids.JobID(jobID))
	if err != nil {
		return Job{}, err
	}
	if err := isolation.RequireProjectMatch(j.ProjectID, ids.ProjectID(projectID)); err != nil {
		return Job{}, err
	}
	return j, nil
}

// JobList lists jobs for the workspace project.
func JobList(dataDir, projectID string) (JobListResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return JobListResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return JobListResult{}, err
	}
	reg, err := jobRegistry(dataDir)
	if err != nil {
		return JobListResult{}, err
	}
	all := reg.List()
	out := make([]Job, 0, len(all))
	for _, j := range all {
		if j.ProjectID == st.Project.ID {
			out = append(out, j)
		}
	}
	return JobListResult{Jobs: out}, nil
}

// JobCancel cancels a pending/running job.
func JobCancel(dataDir, projectID, jobID string) (Job, error) {
	reg, err := jobRegistry(dataDir)
	if err != nil {
		return Job{}, err
	}
	j, err := reg.Get(ids.JobID(jobID))
	if err != nil {
		return Job{}, err
	}
	if err := isolation.RequireProjectMatch(j.ProjectID, ids.ProjectID(projectID)); err != nil {
		return Job{}, err
	}
	return reg.Cancel(ids.JobID(jobID))
}
