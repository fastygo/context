package orchestrator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/agentruntime/orchestrator"
	"github.com/fastygo/context/internal/artifacts/localfs"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	modelfake "github.com/fastygo/context/internal/models/fake"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/retrieval/pack"
	"github.com/fastygo/context/internal/storage/memory"
	toolfake "github.com/fastygo/context/internal/tools/fake"
	toolmem "github.com/fastygo/context/internal/tools/memory"
)

func setup(t *testing.T) (*memory.Store, *localfs.Store, *toolmem.Registry) {
	t.Helper()
	meta := memory.New()
	ctx := context.Background()
	if err := meta.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	arts, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reg := toolmem.NewRegistry()
	if err := reg.Register(toolfake.ReadSnippetSchema()); err != nil {
		t.Fatal(err)
	}
	return meta, arts, reg
}

func samplePack(t *testing.T) retrieval.ContextPack {
	t.Helper()
	built, err := (pack.Builder{}).Build(context.Background(), pack.BuildRequest{
		PackID: "pack1", ProjectID: "p1", TaskID: "t1", PlanID: "plan1",
		Purpose: "agent-run",
		Focus: retrieval.FocusProfile{
			ID: "f1", ProjectID: "p1", Objective: "test",
			RequiredTrustLevel: foundation.TrustProject,
			ContextBudget:      retrieval.Budget{MaxItems: 5, MaxChars: 1000},
		},
		Instructions: []string{"answer with evidence"},
		Items: []pack.DraftItem{{
			ID: "e1", Class: foundation.EvidenceSourceText, Surface: "hello evidence",
			Candidate: retrieval.Candidate{
				ChunkID: "c1", TrustLevel: foundation.TrustProject, TextChecksum: "chk1",
				SourceRef: corpus.SourceRef{
					ProjectID: "p1", SourceID: "s1", ChunkID: "c1",
					Span: foundation.ByteSpan{Start: 0, End: 14}, Checksum: "chk1",
				},
				MergedScore: 1,
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return built
}

func TestAgentRunAllowToolAndReplay(t *testing.T) {
	t.Parallel()
	meta, arts, reg := setup(t)
	longText := strings.Repeat("snippet-body-", 40) // > 256 bytes
	runner := orchestrator.Runner{
		Meta:      meta,
		Artifacts: arts,
		Tools:     reg,
		Exec:      toolfake.ReadSnippetExecutor{Snippets: map[string]string{"s1": longText}},
		Model:     modelfake.Completer{ToolHint: toolfake.ReadSnippetName, ToolInput: `{"source_id":"s1"}`},
	}
	pol := policy.PolicySnapshot{
		ID: "pol1", ProjectID: "p1", Version: "v1",
		Rules: []policy.Rule{{Name: "allow-read", ToolName: toolfake.ReadSnippetName, Decision: policy.DecisionAllow}},
	}
	res, err := runner.Run(context.Background(), orchestrator.Request{
		RunID: "run1", ProjectID: "p1", TaskID: "t1", Owner: "tester",
		Policy: pol, Pack: samplePack(t),
		VerifyFactual: map[string]bool{"e1": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Run.Status != agentruntime.RunStatusCompleted {
		t.Fatalf("status=%s err=%s", res.Run.Status, res.Run.Error)
	}
	if res.ToolCall == nil || res.ToolCall.Status != "completed" {
		t.Fatalf("tool=%#v", res.ToolCall)
	}
	if res.ToolCall.OutputArtifactID == "" {
		t.Fatal("expected long tool output stored as artifact")
	}
	if res.Model.Text == "" {
		t.Fatal("expected model text")
	}

	run, events, err := orchestrator.ReplayRun(context.Background(), meta, "p1", "run1")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != agentruntime.RunStatusCompleted {
		t.Fatalf("replay status=%s", run.Status)
	}
	if len(events) == 0 {
		t.Fatal("expected persisted trace events")
	}
}

func TestDeniedToolCall(t *testing.T) {
	t.Parallel()
	meta, arts, reg := setup(t)
	runner := orchestrator.Runner{
		Meta:      meta,
		Artifacts: arts,
		Tools:     reg,
		Exec:      toolfake.ReadSnippetExecutor{Snippets: map[string]string{"s1": "x"}},
		Model:     modelfake.Completer{},
	}
	pol := policy.PolicySnapshot{
		ID: "pol1", ProjectID: "p1", Version: "v1",
		Rules: []policy.Rule{{Name: "deny-read", ToolName: toolfake.ReadSnippetName, Decision: policy.DecisionDeny}},
	}
	res, err := runner.Run(context.Background(), orchestrator.Request{
		RunID: "run2", ProjectID: "p1", TaskID: "t1",
		Policy: pol, Pack: samplePack(t),
		ToolName: toolfake.ReadSnippetName, ToolInput: []byte(`{"source_id":"s1"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ToolCall == nil || res.ToolCall.Status != "denied" || res.ToolCall.Decision != policy.DecisionDeny {
		t.Fatalf("expected denied tool call, got %#v", res.ToolCall)
	}
	if res.Run.Status != agentruntime.RunStatusCompleted {
		t.Fatalf("denied tool should not fail the whole run by default, status=%s", res.Run.Status)
	}
}

func TestWriteToolNeedsApprovalWithoutRule(t *testing.T) {
	t.Parallel()
	meta, arts, reg := setup(t)
	if err := reg.Register(toolfake.WriteNoteSchema()); err != nil {
		t.Fatal(err)
	}
	runner := orchestrator.Runner{
		Meta: meta, Artifacts: arts, Tools: reg,
		Exec:  toolfake.WriteNoteExecutor{},
		Model: modelfake.Completer{},
	}
	pol := policy.PolicySnapshot{ID: "pol1", ProjectID: "p1", Version: "v1"}
	res, err := runner.Run(context.Background(), orchestrator.Request{
		RunID: "run-ask", ProjectID: "p1", TaskID: "t1",
		Policy: pol, Pack: samplePack(t),
		ToolName: toolfake.WriteNoteName, ToolInput: []byte(`{"text":"x"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ToolCall == nil || res.ToolCall.Status != "needs_approval" || res.ToolCall.Decision != policy.DecisionAsk {
		t.Fatalf("expected needs_approval, got %#v", res.ToolCall)
	}
}
