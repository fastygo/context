package devcli

import (
	"context"
	"encoding/json"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/agentruntime/orchestrator"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/models/factory"
	"github.com/fastygo/context/internal/policy"
	toolfake "github.com/fastygo/context/internal/tools/fake"
	toolmem "github.com/fastygo/context/internal/tools/memory"
	"github.com/fastygo/context/internal/tracing"
)

// AgentRunResult is CLI JSON for agent-run.
type AgentRunResult struct {
	Run            agentruntime.AgentRun `json:"run"`
	PackID         ids.PackID            `json:"pack_id"`
	ModelText      string                `json:"model_text"`
	CompleterKind  string                `json:"completer_kind,omitempty"`
	ModelProvider  string                `json:"model_provider,omitempty"`
	ModelVersion   string                `json:"model_version,omitempty"`
	ToolCall       any                   `json:"tool_call,omitempty"`
	VerifyOK       bool                  `json:"verify_ok"`
	MetaKind       string                `json:"meta_kind,omitempty"`
}

// TraceResult is CLI JSON for trace.
type TraceResult struct {
	Run      agentruntime.AgentRun `json:"run"`
	Events   []tracing.Event       `json:"events"`
	MetaKind string                `json:"meta_kind,omitempty"`
}

// AgentRun builds a pack and executes an agent loop with the configured Completer.
func AgentRun(dataDir, projectID, query, focusID string) (AgentRunResult, error) {
	packRes, err := BuildPack(dataDir, projectID, query, focusID)
	if err != nil {
		return AgentRunResult{}, err
	}
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return AgentRunResult{}, err
	}

	ctx := context.Background()
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return AgentRunResult{}, err
	}
	defer handle.Close()
	meta := handle.Store

	if err := meta.PutProject(ctx, st.Project); err != nil {
		return AgentRunResult{}, err
	}
	if err := meta.PutPack(ctx, packRes.Pack); err != nil {
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

	cfg, err := config.LoadStorageConfigFromEnv()
	if err != nil {
		return AgentRunResult{}, err
	}
	comp, compKind, err := factory.OpenCompleter(cfg, factory.CompleterOptions{
		ToolHint: toolfake.ReadSnippetName, ToolInput: string(input),
	})
	if err != nil {
		return AgentRunResult{}, err
	}

	runner := orchestrator.Runner{
		Meta:      meta,
		Artifacts: arts,
		Tools:     reg,
		Exec:      toolfake.ReadSnippetExecutor{Snippets: snippets},
		Model:     comp,
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
	st.Packs = append(st.Packs, packRes.Pack)
	if err := ws.Save(st); err != nil {
		return AgentRunResult{}, err
	}

	out := AgentRunResult{
		Run:           res.Run,
		PackID:        packRes.Pack.ID,
		ModelText:     res.Model.Text,
		CompleterKind: compKind,
		ModelProvider: res.Model.ModelCall.ProviderID,
		ModelVersion:  res.Model.ModelCall.ModelVersion,
		VerifyOK:      res.Verify.OK,
		MetaKind:      string(handle.Kind),
	}
	if res.ToolCall != nil {
		out.ToolCall = res.ToolCall
	}
	return out, nil
}

// Trace loads a run and its events from postgres metadata when configured,
// otherwise from workspace state.json.
func Trace(dataDir, projectID, runID string) (TraceResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return TraceResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return TraceResult{}, apperr.New(apperr.Validation, "project id mismatch")
	}

	ctx := context.Background()
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return TraceResult{}, err
	}
	defer handle.Close()

	if handle.UsesPostgres() {
		run, err := handle.Store.GetRun(ctx, st.Project.ID, ids.RunID(runID))
		if err != nil {
			return TraceResult{}, err
		}
		events, err := handle.Store.ListTrace(ctx, st.Project.ID, ids.RunID(runID))
		if err != nil {
			return TraceResult{}, err
		}
		return TraceResult{Run: run, Events: events, MetaKind: string(handle.Kind)}, nil
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
	return TraceResult{Run: run, Events: events, MetaKind: string(handle.Kind)}, nil
}
