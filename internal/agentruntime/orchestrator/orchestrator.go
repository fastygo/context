// Package orchestrator runs a minimal agent loop for PoC hypothesis validation.
package orchestrator

import (
	"context"
	"strings"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/policy"
	policyeval "github.com/fastygo/context/internal/policy/eval"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/pack"
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/tools"
	"github.com/fastygo/context/internal/tracing"
)

const LongOutputThreshold = 256

// Request drives one agent run (foreground by default).
type Request struct {
	RunID         ids.RunID
	ProjectID     ids.ProjectID
	TaskID        ids.TaskID
	Owner         string
	Mode          agentruntime.RunMode // empty => foreground
	FocusID       ids.FocusID
	Policy        policy.PolicySnapshot
	Pack          retrieval.ContextPack
	ToolName      string // optional explicit tool after model; empty uses model hint
	ToolInput     []byte
	VerifyFactual map[string]bool
}

// Result captures the completed run and related records for replay/debug.
type Result struct {
	Run       agentruntime.AgentRun
	Pack      retrieval.ContextPack
	Model     models.CompletionResult
	ToolCall  *tools.ToolCall
	Verify    pack.VerifyResult
	Events    []tracing.Event
}

// Runner executes: context pack -> fake model/tool step -> verification.
type Runner struct {
	Meta      storage.MetadataStore
	Artifacts artifacts.ArtifactStore
	Tools     tools.Registry
	Exec      tools.Executor
	Model     models.Completer
	Trace     tracing.Recorder
}

// Run executes one agent loop and persists status transitions.
func (r Runner) Run(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := req.RunID.Validate(); err != nil {
		return Result{}, apperr.Wrap(apperr.Validation, "run_id", err)
	}
	if err := req.ProjectID.Validate(); err != nil {
		return Result{}, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if err := req.Pack.Validate(); err != nil {
		return Result{}, apperr.Wrap(apperr.Validation, "context_pack", err)
	}

	now := time.Now().UTC()
	run := agentruntime.AgentRun{
		ID:        req.RunID,
		ProjectID: req.ProjectID,
		TaskID:    req.TaskID,
		Mode:      agentruntime.RunModeForeground,
		Status:    agentruntime.RunStatusPending,
		FocusID:   "",
		PolicyID:  req.Policy.ID,
		PackID:    req.Pack.ID,
		Owner:     req.Owner,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if req.Mode != "" {
		run.Mode = req.Mode
	}
	if req.FocusID != "" {
		run.FocusID = req.FocusID
	}
	if err := r.persistRun(ctx, run); err != nil {
		return Result{}, err
	}
	var events []tracing.Event
	appendEvent := func(ev tracing.Event) error {
		events = append(events, ev)
		if r.Trace != nil {
			return r.Trace.Append(ctx, ev)
		}
		if store, ok := r.Meta.(storage.TraceStore); ok {
			return store.AppendTrace(ctx, ev)
		}
		return nil
	}

	run.Status = agentruntime.RunStatusRunning
	run.UpdatedAt = time.Now().UTC()
	if err := r.persistRun(ctx, run); err != nil {
		return Result{}, err
	}
	_ = appendEvent(tracing.Event{
		ID: ids.TraceEventID(string(req.RunID) + ":started"), ProjectID: req.ProjectID, RunID: req.RunID,
		Type: tracing.EventRunStarted, Timestamp: time.Now().UTC(),
		Payload: map[string]string{"pack_id": string(req.Pack.ID)},
	})

	if err := r.Meta.PutPack(ctx, req.Pack); err != nil {
		return r.fail(ctx, run, err)
	}

	modelOut, err := r.Model.Complete(ctx, models.CompletionRequest{
		ProjectID: req.ProjectID,
		Pack:      req.Pack,
	})
	if err != nil {
		return r.fail(ctx, run, err)
	}
	modelOut.ModelCall.RunID = req.RunID
	_ = appendEvent(tracing.Event{
		ID: ids.TraceEventID(string(modelOut.ModelCall.ID)), ProjectID: req.ProjectID, RunID: req.RunID,
		Type: tracing.EventModelCall, Timestamp: time.Now().UTC(),
		Payload: map[string]string{
			"provider": modelOut.ModelCall.ProviderID,
			"version":  modelOut.ModelCall.ModelVersion,
			"status":   modelOut.ModelCall.Status,
		},
	})

	var toolCall *tools.ToolCall
	toolName := req.ToolName
	toolInput := req.ToolInput
	if toolName == "" && len(modelOut.ToolHints) > 0 {
		toolName = modelOut.ToolHints[0]
		if len(modelOut.ToolHints) > 1 {
			toolInput = []byte(modelOut.ToolHints[1])
		}
	}
	if toolName != "" {
		tc, err := r.executeTool(ctx, req, toolName, toolInput, &events)
		if err != nil {
			return r.fail(ctx, run, err)
		}
		toolCall = &tc
		if err := r.Meta.PutToolCall(ctx, tc); err != nil {
			return r.fail(ctx, run, err)
		}
	}

	verify, err := (pack.Verifier{}).Verify(ctx, pack.VerifyRequest{
		Pack:           req.Pack,
		TreatAsFactual: req.VerifyFactual,
		Recorder:       eventRecorder{appendEvent},
		RunID:          req.RunID,
	})
	if err != nil {
		return r.fail(ctx, run, err)
	}

	run.Status = agentruntime.RunStatusCompleted
	run.UpdatedAt = time.Now().UTC()
	if err := r.persistRun(ctx, run); err != nil {
		return Result{}, err
	}
	_ = appendEvent(tracing.Event{
		ID: ids.TraceEventID(string(req.RunID) + ":finished"), ProjectID: req.ProjectID, RunID: req.RunID,
		Type: tracing.EventRunFinished, Timestamp: time.Now().UTC(),
		Payload: map[string]string{"status": string(run.Status)},
	})

	return Result{
		Run:      run,
		Pack:     req.Pack,
		Model:    modelOut,
		ToolCall: toolCall,
		Verify:   verify,
		Events:   events,
	}, nil
}

func (r Runner) executeTool(ctx context.Context, req Request, toolName string, input []byte, events *[]tracing.Event) (tools.ToolCall, error) {
	schema, ok := r.Tools.Get(toolName)
	if !ok {
		return tools.ToolCall{}, apperr.New(apperr.NotFound, "tool not registered: "+toolName)
	}
	_ = appendEventHelper(events, r, ctx, tracing.Event{
		ID: ids.TraceEventID(string(req.RunID) + ":tool_reg:" + toolName), ProjectID: req.ProjectID, RunID: req.RunID,
		Type: tracing.EventToolDecision, Timestamp: time.Now().UTC(),
		Payload: map[string]string{"phase": "registered_lookup", "tool": toolName},
	})

	decision, err := (policyeval.Engine{Snapshot: req.Policy, Default: policy.DecisionDeny}).Decide(toolName, schema)
	if err != nil {
		return tools.ToolCall{}, err
	}
	call := tools.ToolCall{
		ID:        ids.ToolCallID(string(req.RunID) + ":tool:" + toolName),
		ProjectID: req.ProjectID,
		RunID:     req.RunID,
		ToolName:  toolName,
		Status:    "pending",
		Decision:  decision,
		RiskLevel: schema.RiskLevel,
	}
	_ = appendEventHelper(events, r, ctx, tracing.Event{
		ID: ids.TraceEventID(string(call.ID) + ":decision"), ProjectID: req.ProjectID, RunID: req.RunID,
		Type: tracing.EventPolicyDecision, Timestamp: time.Now().UTC(),
		Payload: map[string]string{"tool": toolName, "decision": string(decision)},
	})

	if decision == policy.DecisionAsk {
		call.Status = "needs_approval"
		call.Error = "policy requires approval"
		_ = appendEventHelper(events, r, ctx, tracing.Event{
			ID: ids.TraceEventID(string(call.ID) + ":ask"), ProjectID: req.ProjectID, RunID: req.RunID,
			Type: tracing.EventToolDecision, Timestamp: time.Now().UTC(),
			Payload: map[string]string{"tool": toolName, "status": call.Status, "decision": string(decision)},
		})
		return call, nil
	}
	if decision != policy.DecisionAllow {
		call.Status = "denied"
		call.Error = "policy denied tool call"
		_ = appendEventHelper(events, r, ctx, tracing.Event{
			ID: ids.TraceEventID(string(call.ID) + ":denied"), ProjectID: req.ProjectID, RunID: req.RunID,
			Type: tracing.EventToolDecision, Timestamp: time.Now().UTC(),
			Payload: map[string]string{"tool": toolName, "status": call.Status},
		})
		return call, nil
	}

	if r.Exec == nil {
		return tools.ToolCall{}, apperr.New(apperr.Validation, "tool executor required")
	}
	call.Status = "running"
	out, err := r.Exec.Execute(ctx, call, input)
	if err != nil {
		call.Status = "failed"
		call.Error = err.Error()
		return call, nil
	}
	if r.Artifacts != nil && len(out) >= LongOutputThreshold {
		artID := ids.ArtifactID(strings.ReplaceAll(string(call.ID)+"_out", ":", "_"))
		art, err := r.Artifacts.Put(ctx, req.ProjectID, artID, "application/json", out, &artifacts.PutOptions{
			ArtifactType: artifacts.TypeToolOutput,
		})
		if err != nil {
			return tools.ToolCall{}, err
		}
		call.OutputArtifactID = art.ID
		if meta, ok := r.Meta.(storage.ArtifactMetaStore); ok {
			_ = meta.PutArtifactMeta(ctx, art)
		}
	}
	call.Status = "completed"
	_ = appendEventHelper(events, r, ctx, tracing.Event{
		ID: ids.TraceEventID(string(call.ID) + ":executed"), ProjectID: req.ProjectID, RunID: req.RunID,
		Type: tracing.EventToolExecuted, Timestamp: time.Now().UTC(),
		Payload: map[string]string{
			"tool":            toolName,
			"status":          call.Status,
			"output_artifact": string(call.OutputArtifactID),
			"bytes":           itoa(len(out)),
		},
	})
	return call, nil
}

func (r Runner) persistRun(ctx context.Context, run agentruntime.AgentRun) error {
	if r.Meta == nil {
		return apperr.New(apperr.Validation, "metadata store required")
	}
	return r.Meta.PutRun(ctx, run)
}

func (r Runner) fail(ctx context.Context, run agentruntime.AgentRun, err error) (Result, error) {
	run.Status = agentruntime.RunStatusFailed
	run.Error = err.Error()
	run.UpdatedAt = time.Now().UTC()
	_ = r.persistRun(ctx, run)
	return Result{Run: run}, err
}

type eventRecorder struct {
	fn func(tracing.Event) error
}

func (e eventRecorder) Append(ctx context.Context, event tracing.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return e.fn(event)
}

func (e eventRecorder) ListByRun(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) ([]tracing.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

func appendEventHelper(events *[]tracing.Event, r Runner, ctx context.Context, ev tracing.Event) error {
	*events = append(*events, ev)
	if r.Trace != nil {
		return r.Trace.Append(ctx, ev)
	}
	if r.Meta != nil {
		return r.Meta.AppendTrace(ctx, ev)
	}
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ReplayRun loads a completed run and its trace for debugging.
func ReplayRun(ctx context.Context, meta storage.MetadataStore, projectID ids.ProjectID, runID ids.RunID) (agentruntime.AgentRun, []tracing.Event, error) {
	run, err := meta.GetRun(ctx, projectID, runID)
	if err != nil {
		return agentruntime.AgentRun{}, nil, err
	}
	events, err := meta.ListTrace(ctx, projectID, runID)
	if err != nil {
		return agentruntime.AgentRun{}, nil, err
	}
	return run, events, nil
}
