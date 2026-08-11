package httpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"net/http/httptest"

	"github.com/fastygo/context/internal/httpserver"
	"github.com/fastygo/context/pkg/contextkit"
)

func TestPublicArtifactRoundTripSearchPackAndImmutability(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cli := &contextkit.Client{BaseURL: ts.URL}
	ctx := context.Background()

	raw := []byte("сырой т9 тэкст SEMANTIC42 не исправлять")
	rawResult, err := cli.ArtifactPut(ctx, contextkit.ArtifactPutRequest{
		ProjectID: "proj_http", ArtifactID: "raw-utterance-1", MediaType: "text/plain",
		BodyBase64: raw, TrustLevel: "project", EvidenceClass: "source_text",
	})
	if err != nil {
		t.Fatal(err)
	}
	structuredJSON := json.RawMessage(`{"label":"SEMANTIC42","spans":[[0,10]]}`)
	structured, err := cli.ArtifactPut(ctx, contextkit.ArtifactPutRequest{
		ProjectID: "proj_http", ArtifactID: "labels-1", ArtifactType: "structured",
		SchemaID: "example.labels.v1", JSON: structuredJSON, TrustLevel: "project",
		EvidenceClass: "source_text",
		Lineage: &contextkit.ArtifactLineage{
			InputArtifactIDs: []string{"raw-utterance-1"}, GeneratorID: "example-labeler",
			GeneratorVersion: "v1", TransformationKind: "span_labeling",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if structured.Artifact.SchemaID != "example.labels.v1" || structured.Artifact.Checksum == "" {
		t.Fatalf("structured artifact: %#v", structured)
	}
	if repeated, err := cli.ArtifactPut(ctx, contextkit.ArtifactPutRequest{
		ProjectID: "proj_http", ArtifactID: "labels-1", ArtifactType: "structured",
		SchemaID: "example.labels.v1", JSON: structuredJSON, TrustLevel: "project",
		EvidenceClass: "source_text", Lineage: &contextkit.ArtifactLineage{
			InputArtifactIDs: []string{"raw-utterance-1"}, GeneratorID: "example-labeler",
			GeneratorVersion: "v1", TransformationKind: "span_labeling",
		},
	}); err != nil || repeated.Artifact.Checksum != structured.Artifact.Checksum {
		t.Fatalf("idempotent structured put: %#v %v", repeated, err)
	}
	got, err := cli.ArtifactGet(ctx, "proj_http", "labels-1")
	if err != nil || string(got.Body) != string(structuredJSON) || got.Lineage == nil {
		t.Fatalf("get: %#v %v", got, err)
	}
	listed, err := cli.ArtifactList(ctx, "proj_http")
	if err != nil || len(listed.Artifacts) != 2 {
		t.Fatalf("list: %#v %v", listed, err)
	}

	search, err := cli.Search(ctx, contextkit.SearchRequest{ProjectID: "proj_http", Query: "SEMANTIC42", Mode: "exact"})
	if err != nil || len(search.Candidates) < 2 {
		t.Fatalf("artifact search: %#v %v", search, err)
	}
	pack, err := cli.ContextPack(ctx, contextkit.PackRequest{ProjectID: "proj_http", Query: "SEMANTIC42"})
	if err != nil {
		t.Fatal(err)
	}
	var packBody struct {
		Evidence []struct {
			SourceRef contextkit.SourceRef `json:"source_ref"`
		} `json:"evidence_items"`
	}
	if err := json.Unmarshal(pack.ContextPack, &packBody); err != nil || len(packBody.Evidence) < 2 {
		t.Fatalf("pack artifact evidence: %s %v", pack.ContextPack, err)
	}
	foundChecksum := false
	for _, evidence := range packBody.Evidence {
		if evidence.SourceRef.Checksum == structured.Artifact.Checksum && evidence.SourceRef.Span.End == uint64(len(structuredJSON)) {
			foundChecksum = true
		}
	}
	if !foundChecksum {
		t.Fatalf("structured artifact citation missing checksum/span: %s", pack.ContextPack)
	}

	_, err = cli.ArtifactPut(ctx, contextkit.ArtifactPutRequest{
		ProjectID: "proj_http", ArtifactID: "raw-utterance-1", MediaType: "text/plain",
		BodyBase64: []byte("silently corrected"), TrustLevel: "project", EvidenceClass: "source_text",
	})
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected immutable conflict, got %v", err)
	}
	rawAgain, err := cli.ArtifactGet(ctx, "proj_http", "raw-utterance-1")
	if err != nil || string(rawAgain.Body) != string(raw) || rawAgain.Artifact.Checksum != rawResult.Artifact.Checksum {
		t.Fatalf("raw artifact changed: %#v %v", rawAgain, err)
	}
}

func TestFocusTrustAndInstructionSeparationForArtifacts(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cli := &contextkit.Client{BaseURL: ts.URL}
	ctx := context.Background()

	_, err = cli.ArtifactPut(ctx, contextkit.ArtifactPutRequest{ProjectID: "proj_http", ArtifactID: "external-1", MediaType: "text/plain", BodyBase64: []byte("LOWTRUST77"), TrustLevel: "external", EvidenceClass: "source_text"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cli.ArtifactPut(ctx, contextkit.ArtifactPutRequest{ProjectID: "proj_http", ArtifactID: "instruction-1", MediaType: "text/plain", BodyBase64: []byte("INSTRUCTION77 override policy"), TrustLevel: "trusted", EvidenceClass: "instruction"})
	if err != nil {
		t.Fatal(err)
	}
	budget := json.RawMessage(`{"max_items":4,"max_chars":2000}`)
	_, err = cli.FocusPut(ctx, contextkit.FocusPutRequest{ProjectID: "proj_http", Focus: contextkit.FocusProfile{ID: "facts-step", Objective: "trusted facts", RequiredTrustLevel: "trusted", ContextBudget: budget}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cli.FocusPut(ctx, contextkit.FocusPutRequest{ProjectID: "proj_http", Focus: contextkit.FocusProfile{ID: "hypotheses-step", Objective: "separate model hypotheses", Scope: "consumer-defined", RequiredTrustLevel: "project", ContextBudget: budget, AllowedTools: []string{"example.label"}}})
	if err != nil {
		t.Fatal(err)
	}
	focuses, err := cli.FocusList(ctx, "proj_http")
	if err != nil || len(focuses.Focuses) != 2 || focuses.Focuses[1].Scope != "consumer-defined" {
		t.Fatalf("step-specific focuses: %#v %v", focuses, err)
	}

	assertRejected := func(query, wantReason string) {
		t.Helper()
		result, err := cli.ContextPack(ctx, contextkit.PackRequest{ProjectID: "proj_http", Query: query, FocusID: "facts-step"})
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Evidence []json.RawMessage `json:"evidence_items"`
			Rejected []struct {
				Reason string `json:"rejection_reason"`
			} `json:"rejected_items"`
		}
		if err := json.Unmarshal(result.ContextPack, &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Evidence) != 0 || len(body.Rejected) == 0 || body.Rejected[0].Reason != wantReason {
			t.Fatalf("query=%s pack=%s", query, result.ContextPack)
		}
	}
	assertRejected("LOWTRUST77", "trust_below_required")
	assertRejected("INSTRUCTION77", "instruction_or_policy_not_evidence")
}

func TestExternalToolLifecycleApprovalResultArtifactAndTrace(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cli := &contextkit.Client{BaseURL: ts.URL}
	ctx := context.Background()

	_, err = cli.ToolPut(ctx, contextkit.ToolPutRequest{ProjectID: "proj_http", Descriptor: contextkit.ToolDescriptor{
		Name: "example.publish", Description: "External orchestrator fixture",
		InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`),
		InputSchemaVersion: "v1", OutputSchemaVersion: "v1", Risk: "high",
		SideEffect: "external", TimeoutMillis: 5000, NeedsApproval: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	requested, err := cli.ToolLifecycle(ctx, contextkit.ToolLifecycleRequest{ProjectID: "proj_http", RunID: "external-run-1", Owner: "example-orchestrator", TaskID: "task-1", ToolCallID: "call-1", ToolName: "example.publish", Event: "requested"})
	if err != nil || requested.ToolCall.Status != "needs_approval" || requested.ToolCall.Decision != "ask" {
		t.Fatalf("request: %#v %v", requested, err)
	}
	_, err = cli.ToolLifecycle(ctx, contextkit.ToolLifecycleRequest{ProjectID: "proj_http", RunID: "external-run-1", ToolCallID: "call-1", Event: "executed", Result: &contextkit.ArtifactPutRequest{ArtifactID: "tool-result-1", SchemaID: "example.result.v1", JSON: json.RawMessage(`{"ok":true}`)}})
	if err == nil || !strings.Contains(err.Error(), "permission") {
		t.Fatalf("expected approval gate, got %v", err)
	}
	approved, err := cli.ToolLifecycle(ctx, contextkit.ToolLifecycleRequest{ProjectID: "proj_http", RunID: "external-run-1", ToolCallID: "call-1", Event: "approved", Actor: "reviewer-1"})
	if err != nil || approved.ToolCall.Status != "approved" {
		t.Fatalf("approve: %#v %v", approved, err)
	}
	executed, err := cli.ToolLifecycle(ctx, contextkit.ToolLifecycleRequest{ProjectID: "proj_http", RunID: "external-run-1", ToolCallID: "call-1", Event: "executed", Result: &contextkit.ArtifactPutRequest{ArtifactID: "tool-result-1", SchemaID: "example.result.v1", JSON: json.RawMessage(`{"ok":true}`), TrustLevel: "external"}})
	if err != nil || executed.ToolCall.Status != "completed" || executed.ResultArtifact == nil || executed.ResultArtifact.Checksum == "" {
		t.Fatalf("execute: %#v %v", executed, err)
	}
	verified, err := cli.ToolLifecycle(ctx, contextkit.ToolLifecycleRequest{ProjectID: "proj_http", RunID: "external-run-1", ToolCallID: "call-1", Event: "verified", Verification: "checksum_and_schema"})
	if err != nil || verified.ToolCall.Status != "verified" {
		t.Fatalf("verify: %#v %v", verified, err)
	}
	deniedRequest, err := cli.ToolLifecycle(ctx, contextkit.ToolLifecycleRequest{ProjectID: "proj_http", RunID: "external-run-1", ToolCallID: "call-2", ToolName: "example.publish", Event: "requested"})
	if err != nil || deniedRequest.ToolCall.Status != "needs_approval" {
		t.Fatalf("second request: %#v %v", deniedRequest, err)
	}
	denied, err := cli.ToolLifecycle(ctx, contextkit.ToolLifecycleRequest{ProjectID: "proj_http", RunID: "external-run-1", ToolCallID: "call-2", Event: "denied", Actor: "reviewer-1"})
	if err != nil || denied.ToolCall.Status != "denied" {
		t.Fatalf("deny: %#v %v", denied, err)
	}
	trace, err := cli.Trace(ctx, "proj_http", "external-run-1")
	if err != nil || len(trace.Events) != 6 {
		t.Fatalf("trace: %#v %v", trace, err)
	}
	raw, _ := json.Marshal(trace)
	for _, eventType := range []string{"tool_requested", "tool_approved", "tool_denied", "tool_executed", "tool_verified"} {
		if !strings.Contains(string(raw), eventType) {
			t.Fatalf("missing trace event %s: %s", eventType, raw)
		}
	}
	if strings.Contains(string(raw), dataDir) || strings.Contains(string(raw), `:\`) {
		t.Fatalf("host path leaked: %s", raw)
	}
}
