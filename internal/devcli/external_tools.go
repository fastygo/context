package devcli

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/policy"
	policyeval "github.com/fastygo/context/internal/policy/eval"
	"github.com/fastygo/context/internal/policy/isolation"
	"github.com/fastygo/context/internal/tools"
	"github.com/fastygo/context/internal/tracing"
)

const (
	ToolEventRequested = "requested"
	ToolEventApproved  = "approved"
	ToolEventDenied    = "denied"
	ToolEventExecuted  = "executed"
	ToolEventVerified  = "verified"
)

// ExternalToolDescriptor is the brand-neutral public form of ToolSchema.
type ExternalToolDescriptor struct {
	Name                string                `json:"name"`
	Description         string                `json:"description,omitempty"`
	InputSchema         json.RawMessage       `json:"input_schema"`
	OutputSchema        json.RawMessage       `json:"output_schema"`
	InputSchemaVersion  string                `json:"input_schema_version"`
	OutputSchemaVersion string                `json:"output_schema_version"`
	Permission          string                `json:"permission"`
	Risk                policy.RiskLevel      `json:"risk"`
	SideEffect          tools.SideEffectClass `json:"side_effect"`
	TimeoutMillis       int64                 `json:"timeout_millis"`
	NeedsApproval       bool                  `json:"needs_approval,omitempty"`
}

func (d ExternalToolDescriptor) schema() tools.ToolSchema {
	return tools.ToolSchema{
		Name: d.Name, Description: d.Description,
		InputSchemaJSON: string(d.InputSchema), OutputSchemaJSON: string(d.OutputSchema),
		InputSchemaVer: d.InputSchemaVersion, OutputSchemaVer: d.OutputSchemaVersion,
		PermissionPolicy: d.Permission, RiskLevel: d.Risk, SideEffectClass: d.SideEffect,
		TimeoutMillis: d.TimeoutMillis, NeedsApproval: d.NeedsApproval,
	}
}

func descriptorView(s tools.ToolSchema) ExternalToolDescriptor {
	return ExternalToolDescriptor{
		Name: s.Name, Description: s.Description,
		InputSchema: json.RawMessage(s.InputSchemaJSON), OutputSchema: json.RawMessage(s.OutputSchemaJSON),
		InputSchemaVersion: s.InputSchemaVer, OutputSchemaVersion: s.OutputSchemaVer,
		Permission: s.PermissionPolicy, Risk: s.RiskLevel, SideEffect: s.SideEffectClass,
		TimeoutMillis: s.TimeoutMillis, NeedsApproval: s.NeedsApproval,
	}
}

type ToolDescriptorResult struct {
	Descriptor ExternalToolDescriptor `json:"descriptor"`
}

type ToolDescriptorListResult struct {
	Descriptors []ExternalToolDescriptor `json:"descriptors"`
}

func PutToolDescriptor(dataDir, projectID string, descriptor ExternalToolDescriptor) (ToolDescriptorResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ToolDescriptorResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return ToolDescriptorResult{}, err
	}
	schema := descriptor.schema()
	if err := schema.Validate(); err != nil {
		return ToolDescriptorResult{}, apperr.Wrap(apperr.Validation, "tool descriptor", err)
	}
	if schema.PermissionPolicy != "" {
		if err := policy.Decision(schema.PermissionPolicy).Validate(); err != nil {
			return ToolDescriptorResult{}, apperr.Wrap(apperr.Validation, "tool permission", err)
		}
	}
	for i, existing := range st.ToolSchemas {
		if existing.Name == schema.Name {
			st.ToolSchemas[i] = schema
			if err := ws.Save(st); err != nil {
				return ToolDescriptorResult{}, err
			}
			return ToolDescriptorResult{Descriptor: descriptorView(schema)}, nil
		}
	}
	st.ToolSchemas = append(st.ToolSchemas, schema)
	if err := ws.Save(st); err != nil {
		return ToolDescriptorResult{}, err
	}
	return ToolDescriptorResult{Descriptor: descriptorView(schema)}, nil
}

func ListToolDescriptors(dataDir, projectID string) (ToolDescriptorListResult, error) {
	st, err := (Workspace{DataDir: dataDir}).Load()
	if err != nil {
		return ToolDescriptorListResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(projectID)); err != nil {
		return ToolDescriptorListResult{}, err
	}
	out := make([]ExternalToolDescriptor, 0, len(st.ToolSchemas))
	for _, schema := range st.ToolSchemas {
		out = append(out, descriptorView(schema))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return ToolDescriptorListResult{Descriptors: out}, nil
}

// ToolLifecycleInput records an orchestrator-side lifecycle transition. Core
// evaluates policy and stores results but never executes arbitrary binaries.
type ToolLifecycleInput struct {
	ProjectID       string
	RunID           string
	Owner           string
	TaskID          string
	ToolCallID      string
	ToolName        string
	Event           string
	InputArtifactID string
	Result          *ArtifactPutInput
	Actor           string
	Verification    string
}

type ToolCallView struct {
	ID               ids.ToolCallID   `json:"tool_call_id"`
	ProjectID        ids.ProjectID    `json:"project_id"`
	RunID            ids.RunID        `json:"run_id"`
	ToolName         string           `json:"tool_name"`
	InputArtifactID  ids.ArtifactID   `json:"input_artifact_id,omitempty"`
	OutputArtifactID ids.ArtifactID   `json:"output_artifact_id,omitempty"`
	Status           string           `json:"status"`
	Decision         policy.Decision  `json:"decision"`
	Risk             policy.RiskLevel `json:"risk"`
	Error            string           `json:"error,omitempty"`
}

type ToolLifecycleResult struct {
	ToolCall ToolCallView  `json:"tool_call"`
	Artifact *ArtifactView `json:"result_artifact,omitempty"`
}

func RecordToolLifecycle(ctx context.Context, dataDir string, in ToolLifecycleInput) (ToolLifecycleResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return ToolLifecycleResult{}, err
	}
	if err := isolation.RequireProjectMatch(st.Project.ID, ids.ProjectID(in.ProjectID)); err != nil {
		return ToolLifecycleResult{}, err
	}
	if strings.TrimSpace(in.Event) == "" || strings.TrimSpace(in.RunID) == "" || strings.TrimSpace(in.ToolCallID) == "" {
		return ToolLifecycleResult{}, apperr.New(apperr.Validation, "event, run_id, and tool_call_id required")
	}

	if in.Event == ToolEventRequested {
		return requestTool(ctx, ws, st, in)
	}
	call, index, ok := findToolCall(st, ids.ToolCallID(in.ToolCallID))
	if !ok || call.RunID != ids.RunID(in.RunID) {
		return ToolLifecycleResult{}, apperr.New(apperr.NotFound, "tool call not found")
	}
	var resultArtifact *ArtifactView
	var eventType tracing.EventType
	switch in.Event {
	case ToolEventApproved:
		if call.Status != "needs_approval" {
			return ToolLifecycleResult{}, apperr.New(apperr.Conflict, "only a needs_approval tool call can be approved")
		}
		if strings.TrimSpace(in.Actor) == "" {
			return ToolLifecycleResult{}, apperr.New(apperr.Validation, "actor required for approved event")
		}
		call.Status, call.Decision, call.Error = "approved", policy.DecisionAllow, ""
		eventType = tracing.EventToolApproved
	case ToolEventDenied:
		if call.Status != "needs_approval" && call.Status != "approved" {
			return ToolLifecycleResult{}, apperr.New(apperr.Conflict, "tool call cannot be denied from current status")
		}
		if strings.TrimSpace(in.Actor) == "" {
			return ToolLifecycleResult{}, apperr.New(apperr.Validation, "actor required for denied event")
		}
		call.Status, call.Decision, call.Error = "denied", policy.DecisionDeny, "approval denied"
		eventType = tracing.EventToolDenied
	case ToolEventExecuted:
		if call.Status != "approved" {
			return ToolLifecycleResult{}, apperr.New(apperr.Permission, "tool call must be approved before execution is recorded")
		}
		if in.Result == nil {
			return ToolLifecycleResult{}, apperr.New(apperr.Validation, "result artifact required for executed event")
		}
		artifactIn := *in.Result
		artifactIn.ProjectID = in.ProjectID
		artifactIn.EvidenceClass = foundation.EvidenceToolOutput
		if artifactIn.Lineage != nil {
			lineage := *artifactIn.Lineage
			lineage.ToolCallID = call.ID
			artifactIn.Lineage = &lineage
		} else if call.InputArtifactID != "" {
			schema, _ := findToolSchema(st, call.ToolName)
			artifactIn.Lineage = &artifacts.ArtifactLineage{
				InputArtifactIDs: []ids.ArtifactID{call.InputArtifactID}, ToolCallID: call.ID,
				GeneratorID: call.ToolName, GeneratorVersion: schema.OutputSchemaVer,
				TransformationKind: "tool_result",
			}
		}
		if artifactIn.SchemaID == "" {
			artifactIn.ArtifactType = artifacts.TypeToolOutput
		} else {
			artifactIn.ArtifactType = artifacts.TypeStructured
		}
		artifactResult, err := PutArtifact(ctx, dataDir, artifactIn)
		if err != nil {
			return ToolLifecycleResult{}, err
		}
		st, err = ws.Load()
		if err != nil {
			return ToolLifecycleResult{}, err
		}
		call, index, ok = findToolCall(st, ids.ToolCallID(in.ToolCallID))
		if !ok {
			return ToolLifecycleResult{}, apperr.New(apperr.Conflict, "tool call disappeared while storing result")
		}
		call.Status = "completed"
		call.OutputArtifactID = artifactResult.Artifact.ID
		resultArtifact = &artifactResult.Artifact
		eventType = tracing.EventToolExecuted
	case ToolEventVerified:
		if call.Status != "completed" {
			return ToolLifecycleResult{}, apperr.New(apperr.Conflict, "only a completed tool call can be verified")
		}
		if strings.TrimSpace(in.Verification) == "" {
			return ToolLifecycleResult{}, apperr.New(apperr.Validation, "verification required for verified event")
		}
		call.Status = "verified"
		eventType = tracing.EventToolVerified
	default:
		return ToolLifecycleResult{}, apperr.New(apperr.Validation, "event must be requested|approved|denied|executed|verified")
	}
	st.ToolCalls[index] = call
	event := toolTraceEvent(call, eventType, in.Verification)
	if in.Actor != "" {
		event.Payload["actor"] = in.Actor
	}
	if err := appendTraceEvent(&st, event); err != nil {
		return ToolLifecycleResult{}, err
	}
	if err := ws.Save(st); err != nil {
		return ToolLifecycleResult{}, err
	}
	if err := persistExternalTool(ctx, st, call, []tracing.Event{event}); err != nil {
		return ToolLifecycleResult{}, err
	}
	return ToolLifecycleResult{ToolCall: toolCallView(call), Artifact: resultArtifact}, nil
}

func requestTool(ctx context.Context, ws Workspace, st State, in ToolLifecycleInput) (ToolLifecycleResult, error) {
	if strings.TrimSpace(in.ToolName) == "" {
		return ToolLifecycleResult{}, apperr.New(apperr.Validation, "tool_name required for requested event")
	}
	if _, _, exists := findToolCall(st, ids.ToolCallID(in.ToolCallID)); exists {
		return ToolLifecycleResult{}, apperr.New(apperr.Conflict, "tool call already exists")
	}
	schema, ok := findToolSchema(st, in.ToolName)
	if !ok {
		return ToolLifecycleResult{}, apperr.New(apperr.NotFound, "tool descriptor not registered")
	}
	run, runIndex, runExists := findRun(st, ids.RunID(in.RunID))
	if !runExists {
		if strings.TrimSpace(in.Owner) == "" {
			return ToolLifecycleResult{}, apperr.New(apperr.Validation, "owner required when registering an external run")
		}
		now := time.Now().UTC()
		run = agentruntime.AgentRun{ID: ids.RunID(in.RunID), ProjectID: st.Project.ID, TaskID: ids.TaskID(in.TaskID), Mode: agentruntime.RunModeForeground, Status: agentruntime.RunStatusRunning, Owner: in.Owner, CreatedAt: now, UpdatedAt: now}
		if err := run.Validate(); err != nil {
			return ToolLifecycleResult{}, apperr.Wrap(apperr.Validation, "external run", err)
		}
		st.Runs = append(st.Runs, run)
		runIndex = len(st.Runs) - 1
	}

	decision, err := descriptorDecision(st.Project.ID, schema)
	if err != nil {
		return ToolLifecycleResult{}, err
	}
	call := tools.ToolCall{ID: ids.ToolCallID(in.ToolCallID), ProjectID: st.Project.ID, RunID: run.ID, ToolName: schema.Name, InputArtifactID: ids.ArtifactID(in.InputArtifactID), Decision: decision, RiskLevel: schema.RiskLevel}
	switch decision {
	case policy.DecisionAllow:
		call.Status = "approved"
	case policy.DecisionAsk:
		call.Status, call.Error = "needs_approval", "policy requires approval"
	case policy.DecisionDeny:
		call.Status, call.Error = "denied", "policy denied tool call"
	}
	if err := call.Validate(); err != nil {
		return ToolLifecycleResult{}, apperr.Wrap(apperr.Validation, "tool call", err)
	}
	st.ToolCalls = append(st.ToolCalls, call)
	st.Runs[runIndex] = run
	requested := toolTraceEvent(call, tracing.EventToolRequested, "")
	requested.Payload["input_schema_version"] = schema.InputSchemaVer
	requested.Payload["output_schema_version"] = schema.OutputSchemaVer
	requested.Payload["side_effect"] = string(schema.SideEffectClass)
	requested.Payload["timeout_millis"] = strconv.FormatInt(schema.TimeoutMillis, 10)
	requested.Payload["needs_approval"] = strconv.FormatBool(schema.NeedsApproval)
	events := []tracing.Event{requested}
	if decision == policy.DecisionAllow {
		events = append(events, toolTraceEvent(call, tracing.EventToolApproved, "policy"))
	} else if decision == policy.DecisionDeny {
		events = append(events, toolTraceEvent(call, tracing.EventToolDenied, "policy"))
	}
	for _, event := range events {
		if err := appendTraceEvent(&st, event); err != nil {
			return ToolLifecycleResult{}, err
		}
	}
	if err := ws.Save(st); err != nil {
		return ToolLifecycleResult{}, err
	}
	if err := persistExternalTool(ctx, st, call, events); err != nil {
		return ToolLifecycleResult{}, err
	}
	return ToolLifecycleResult{ToolCall: toolCallView(call)}, nil
}

func descriptorDecision(projectID ids.ProjectID, schema tools.ToolSchema) (policy.Decision, error) {
	snapshot := policy.PolicySnapshot{ID: "external-tool-policy", ProjectID: projectID, Version: "v1"}
	if schema.PermissionPolicy != "" {
		snapshot.Rules = []policy.Rule{{Name: "descriptor-permission", ToolName: schema.Name, Decision: policy.Decision(schema.PermissionPolicy)}}
	}
	if schema.NeedsApproval {
		return policy.DecisionAsk, nil
	}
	return (policyeval.Engine{Snapshot: snapshot, Default: policy.DecisionDeny}).Decide(schema.Name, schema)
}

func toolTraceEvent(call tools.ToolCall, eventType tracing.EventType, detail string) tracing.Event {
	payload := map[string]string{"tool_call_id": string(call.ID), "tool": call.ToolName, "status": call.Status, "decision": string(call.Decision)}
	if call.OutputArtifactID != "" {
		payload["output_artifact_id"] = string(call.OutputArtifactID)
	}
	if detail != "" {
		payload["detail"] = detail
	}
	return tracing.Event{ID: ids.TraceEventID(string(call.ID) + ":" + string(eventType)), ProjectID: call.ProjectID, RunID: call.RunID, Type: eventType, Timestamp: time.Now().UTC(), Payload: payload}
}

func appendTraceEvent(st *State, event tracing.Event) error {
	if err := event.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "trace event", err)
	}
	for _, existing := range st.Traces {
		if existing.ID != event.ID {
			continue
		}
		if existing.ProjectID == event.ProjectID && existing.RunID == event.RunID && existing.Type == event.Type && reflect.DeepEqual(existing.Payload, event.Payload) {
			return nil
		}
		return apperr.New(apperr.Conflict, "trace event id already exists")
	}
	st.Traces = append(st.Traces, event)
	return nil
}

func persistExternalTool(ctx context.Context, st State, call tools.ToolCall, events []tracing.Event) error {
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return err
	}
	defer handle.Close()
	if !handle.UsesPostgres() {
		return nil
	}
	if err := handle.Store.PutProject(ctx, st.Project); err != nil {
		return err
	}
	run, _, ok := findRun(st, call.RunID)
	if !ok {
		return apperr.New(apperr.NotFound, "external run not found")
	}
	if err := handle.Store.PutRun(ctx, run); err != nil {
		return err
	}
	if err := handle.Store.PutToolCall(ctx, call); err != nil {
		return err
	}
	for _, event := range events {
		if err := handle.Store.AppendTrace(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func findToolSchema(st State, name string) (tools.ToolSchema, bool) {
	for _, schema := range st.ToolSchemas {
		if schema.Name == name {
			return schema, true
		}
	}
	return tools.ToolSchema{}, false
}

func findToolCall(st State, id ids.ToolCallID) (tools.ToolCall, int, bool) {
	for i, call := range st.ToolCalls {
		if call.ID == id {
			return call, i, true
		}
	}
	return tools.ToolCall{}, -1, false
}

func findRun(st State, id ids.RunID) (agentruntime.AgentRun, int, bool) {
	for i, run := range st.Runs {
		if run.ID == id {
			return run, i, true
		}
	}
	return agentruntime.AgentRun{}, -1, false
}

func toolCallView(call tools.ToolCall) ToolCallView {
	return ToolCallView{ID: call.ID, ProjectID: call.ProjectID, RunID: call.RunID, ToolName: call.ToolName, InputArtifactID: call.InputArtifactID, OutputArtifactID: call.OutputArtifactID, Status: call.Status, Decision: call.Decision, Risk: call.RiskLevel, Error: call.Error}
}
