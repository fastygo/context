package devcli

import (
	"context"
	"encoding/json"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/agentruntime/orchestrator"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/ids"
	modelfake "github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/storage/memory"
	toolfake "github.com/fastygo/context/internal/tools/fake"
	toolmem "github.com/fastygo/context/internal/tools/memory"
	"github.com/fastygo/context/internal/tracing"
)

// AgentRunResult is CLI JSON for agent-run.
type AgentRunResult struct {
	Run      agentruntime.AgentRun `json:"run"`
	PackID   ids.PackID            `json:"pack_id"`
	ModelText string               `json:"model_text"`
	ToolCall any                   `json:"tool_call,omitempty"`
	VerifyOK bool                  `json:"verify_ok"`
}

// TraceResult is CLI JSON for trace.
type TraceResult struct {
	Run    agentruntime.AgentRun `json:"run"`
	Events []tracing.Event       `json:"events"`
}

// AgentRun builds a pack and executes a fake agent loop.
func AgentRun(dataDir, projectID, query string) (AgentRunResult, error) {
	packRes, err := BuildPack(dataDir, projectID, query)
	if err != nil {
		return AgentRunResult{}, err
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return AgentRunResult{}, err
	}

	meta := memory.New()
	ctx := context.Background()
	if err := meta.PutProject(ctx, st.Project); err != nil {
		return AgentRunResult{}, err
	}
	arts, err := localfs.New(ws.ArtifactsDir())
	if err != nil {
		return AgentRunResult{}, err
	}
	reg := toolmem.NewRegistry()
	_ = reg.Register(toolfake.ReadSnippetSchema())

	snippets := map[string]string{}
	for _, ch := range st.Chunks {
		snippets[string(ch.SourceID)] = ch.Text
	}
	sourceID := "s1"
	if len(packRes.Pack.EvidenceItems) > 0 {
		sourceID = string(packRes.Pack.EvidenceItems[0].SourceRef.SourceID)
	}
	input, _ := json.Marshal(map[string]string{"source_id": sourceID})

	runner := orchestrator.Runner{
		Meta:      meta,
		Artifacts: arts,
		Tools:     reg,
		Exec:      toolfake.ReadSnippetExecutor{Snippets: snippets},
		Model:     modelfake.Completer{ToolHint: toolfake.ReadSnippetName, ToolInput: string(input)},
	}
	pol := policy.PolicySnapshot{
		ID: "cli-policy", ProjectID: st.Project.ID, Version: "v1",
		Rules: []policy.Rule{{
			Name: "allow-read", ToolName: toolfake.ReadSnippetName, Decision: policy.DecisionAllow,
		}},
	}
	runID := ids.RunID("run_" + string(packRes.Pack.ID))
	factual := map[string]bool{}
	for _, e := range packRes.Pack.EvidenceItems {
		factual[e.ID] = true
	}
	res, err := runner.Run(ctx, orchestrator.Request{
		RunID: runID, ProjectID: st.Project.ID, TaskID: "cli-task", Owner: "cli",
		Policy: pol, Pack: packRes.Pack, VerifyFactual: factual,
	})
	if err != nil {
		return AgentRunResult{}, err
	}

	st.Runs = append(st.Runs, res.Run)
	if res.ToolCall != nil {
		st.ToolCalls = append(st.ToolCalls, *res.ToolCall)
	}
	st.Traces = append(st.Traces, res.Events...)
	if err := ws.Save(st); err != nil {
		return AgentRunResult{}, err
	}

	out := AgentRunResult{
		Run:       res.Run,
		PackID:    packRes.Pack.ID,
		ModelText: res.Model.Text,
		VerifyOK:  res.Verify.OK,
	}
	if res.ToolCall != nil {
		out.ToolCall = res.ToolCall
	}
	return out, nil
}

// Trace loads a run and its events from workspace state.
func Trace(dataDir, projectID, runID string) (TraceResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return TraceResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return TraceResult{}, apperr.New(apperr.Validation, "project id mismatch")
	}
	var run agentruntime.AgentRun
	found := false
	for _, r := range st.Runs {
		if string(r.ID) == runID {
			run = r
			found = true
			break
		}
	}
	if !found {
		return TraceResult{}, apperr.New(apperr.NotFound, "run not found")
	}
	var events []tracing.Event
	for _, ev := range st.Traces {
		if string(ev.RunID) == runID {
			events = append(events, ev)
		}
	}
	return TraceResult{Run: run, Events: events}, nil
}
