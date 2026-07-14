package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/storage/memory"
	"github.com/fastygo/context/internal/tools"
	"github.com/fastygo/context/internal/tracing"
)

func TestProjectSourceChunkRoundTrip(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()

	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	src := corpus.Source{
		ID:         "s1",
		ProjectID:  "p1",
		Type:       corpus.SourceTypeFile,
		PathKey:    "readme",
		TrustLevel: foundation.TrustProject,
	}
	if err := store.PutSource(ctx, src); err != nil {
		t.Fatal(err)
	}
	ch := corpus.Chunk{
		ID:             "c1",
		ProjectID:      "p1",
		SourceID:       "s1",
		ArtifactID:     "a1",
		SnapshotID:     "snap1",
		ChunkerVersion: "para-v1",
		Span:           foundation.ByteSpan{Start: 0, End: 4},
		TextChecksum:   "aa",
		ChunkHash:      "bb",
	}
	if err := store.PutChunk(ctx, ch); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetChunk(ctx, "p1", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ChunkHash != "bb" {
		t.Fatalf("chunk=%#v", got)
	}
}

func TestListProjectsSorted(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	for _, id := range []string{"p-c", "p-a", "p-b"} {
		if err := store.PutProject(ctx, corpus.Project{ID: ids.ProjectID(id), Name: id}); err != nil {
			t.Fatal(err)
		}
	}
	projects, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 3 || projects[0].ID != "p-a" || projects[1].ID != "p-b" || projects[2].ID != "p-c" {
		t.Fatalf("order=%v", projectIDs(projects))
	}
}

func TestSetActiveSnapshotRequiresReady(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	building := indexing.IndexSnapshot{
		ID:        "snap1",
		ProjectID: "p1",
		Status:    foundation.SnapshotBuilding,
	}
	if err := store.PutSnapshot(ctx, building); err != nil {
		t.Fatal(err)
	}
	if err := store.SetActiveSnapshot(ctx, "p1", "snap1"); !apperr.Is(err, apperr.Conflict) {
		t.Fatalf("expected conflict, got %v", err)
	}

	ready := indexing.IndexSnapshot{
		ID:               "snap2",
		ProjectID:        "p1",
		Status:           foundation.SnapshotReady,
		SourceMerkleRoot: "aa",
		ChunkSetHash:     "bb",
		ParserVersion:    "p-v1",
		ChunkerVersion:   "c-v1",
	}
	if err := store.PutSnapshot(ctx, ready); err != nil {
		t.Fatal(err)
	}
	if err := store.SetActiveSnapshot(ctx, "p1", "snap2"); err != nil {
		t.Fatal(err)
	}
	project, err := store.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if project.ActiveSnapshotID != "snap2" {
		t.Fatalf("active=%s", project.ActiveSnapshotID)
	}
}

func TestTraceAppendOrderAndConflict(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	ts := time.Unix(10, 0).UTC()
	e1 := tracing.Event{ID: "e1", ProjectID: "p1", RunID: "r1", Type: tracing.EventRunStarted, Timestamp: ts}
	e2 := tracing.Event{ID: "e2", ProjectID: "p1", RunID: "r1", Type: tracing.EventRunFinished, Timestamp: ts.Add(time.Second)}
	if err := store.AppendTrace(ctx, e1); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendTrace(ctx, e2); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendTrace(ctx, e1); !apperr.Is(err, apperr.Conflict) {
		t.Fatalf("expected conflict on duplicate event id, got %v", err)
	}
	events, err := store.ListTrace(ctx, "p1", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].ID != "e1" || events[1].ID != "e2" {
		t.Fatalf("events=%v", events)
	}
}

func TestArtifactMetaAndPackRunTool(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	art := artifacts.Artifact{
		ID:         "a1",
		ProjectID:  "p1",
		MediaType:  "text/plain",
		ByteSize:   1,
		Checksum:   "aa",
		StorageURI: "localfs://p1/a1",
	}
	if err := store.PutArtifactMeta(ctx, art); err != nil {
		t.Fatal(err)
	}
	listed, err := store.ListArtifacts(ctx, "p1")
	if err != nil || len(listed) != 1 {
		t.Fatalf("listed=%v err=%v", listed, err)
	}

	pack := retrieval.ContextPack{
		ID:              "pack1",
		ProjectID:       "p1",
		RetrievalPlanID: "plan1",
		Purpose:         "test",
	}
	if err := store.PutPack(ctx, pack); err != nil {
		t.Fatal(err)
	}
	run := agentruntime.AgentRun{
		ID:        "run1",
		ProjectID: "p1",
		Mode:      agentruntime.RunModeForeground,
		Status:    agentruntime.RunStatusPending,
	}
	if err := store.PutRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	call := tools.ToolCall{
		ID:        "tc1",
		ProjectID: "p1",
		ToolName:  "noop",
		Decision:  policy.DecisionAllow,
	}
	if err := store.PutToolCall(ctx, call); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetToolCall(ctx, "p1", "tc1"); err != nil {
		t.Fatal(err)
	}
}

func TestNotFoundErrors(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if _, err := store.GetProject(ctx, "missing"); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestArtifactLineageRoundTripAndDeterministicOrder(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []ids.ArtifactID{"input-a", "input-b", "output-z", "output-a"} {
		art := artifacts.Artifact{
			ID:           id,
			ProjectID:    "p1",
			MediaType:    "application/json",
			ByteSize:     2,
			Checksum:     foundation.ChecksumHex("checksum-" + string(id)),
			StorageURI:   "localfs://p1/" + string(id),
			ArtifactType: artifacts.TypeStructured,
			SchemaID:     "example.output.v1",
		}
		if err := store.PutArtifactMeta(ctx, art); err != nil {
			t.Fatal(err)
		}
	}
	for _, outputID := range []ids.ArtifactID{"output-z", "output-a"} {
		lineage := artifacts.ArtifactLineage{
			ProjectID:          "p1",
			OutputArtifactID:   outputID,
			InputArtifactIDs:   []ids.ArtifactID{"input-a", "input-b"},
			GeneratorID:        "example-generator",
			GeneratorVersion:   "v1",
			TransformationKind: "aggregate",
			CreatedAt:          time.Unix(10, 0).UTC(),
		}
		if err := store.PutArtifactLineage(ctx, lineage); err != nil {
			t.Fatal(err)
		}
		lineage.InputArtifactIDs[0] = "caller-mutated"
	}

	lineages, err := store.ListArtifactLineage(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(lineages) != 2 ||
		lineages[0].OutputArtifactID != "output-a" ||
		lineages[1].OutputArtifactID != "output-z" {
		t.Fatalf("lineage order=%#v", lineages)
	}
	got, err := store.GetArtifactLineage(ctx, "p1", "output-z")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.InputArtifactIDs) != 2 || got.InputArtifactIDs[0] != "input-a" {
		t.Fatalf("lineage inputs=%v", got.InputArtifactIDs)
	}
	got.InputArtifactIDs[0] = "returned-mutated"
	again, err := store.GetArtifactLineage(ctx, "p1", "output-z")
	if err != nil {
		t.Fatal(err)
	}
	if again.InputArtifactIDs[0] != "input-a" {
		t.Fatalf("stored lineage was mutated through returned slice: %v", again.InputArtifactIDs)
	}
}

func TestArtifactLineageRequiresStoredProjectAndArtifacts(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	lineage := artifacts.ArtifactLineage{
		ProjectID:          "p1",
		OutputArtifactID:   "output",
		InputArtifactIDs:   []ids.ArtifactID{"input"},
		GeneratorID:        "example-generator",
		GeneratorVersion:   "v1",
		TransformationKind: "aggregate",
		CreatedAt:          time.Unix(10, 0).UTC(),
	}
	if err := store.PutArtifactLineage(ctx, lineage); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("expected missing project error, got %v", err)
	}
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutArtifactLineage(ctx, lineage); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("expected missing output artifact error, got %v", err)
	}
	output := artifacts.Artifact{
		ID:           "output",
		ProjectID:    "p1",
		MediaType:    "application/json",
		ByteSize:     2,
		Checksum:     "output-checksum",
		StorageURI:   "localfs://p1/output",
		ArtifactType: artifacts.TypeStructured,
		SchemaID:     "example.output.v1",
	}
	if err := store.PutArtifactMeta(ctx, output); err != nil {
		t.Fatal(err)
	}
	if err := store.PutArtifactLineage(ctx, lineage); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("expected missing input artifact error, got %v", err)
	}
}

func TestTemporalSourceDoesNotCreateRuntimeTrace(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	base := time.Unix(100, 0).UTC()
	src := corpus.Source{
		ID:         "event-source",
		ProjectID:  "p1",
		Type:       corpus.SourceTypeArtifact,
		PathKey:    "events/batch-1",
		TrustLevel: foundation.TrustProject,
		TemporalMetadata: &corpus.TemporalMetadata{
			Range: corpus.TemporalRange{
				Start: base,
				End:   base.Add(time.Minute),
				Basis: corpus.TimeBasisOccurred,
			},
			IngestedAt: base.Add(2 * time.Minute),
		},
	}
	if err := store.PutSource(ctx, src); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListTrace(ctx, "p1", "source-events-are-not-runs")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("source ingest must not append runtime traces: %#v", events)
	}
}

func TestTombstoneSourceExcludesFromLiveListSemantics(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	src := corpus.Source{
		ID: "s1", ProjectID: "p1", Type: corpus.SourceTypeFile,
		PathKey: "a.txt", TrustLevel: foundation.TrustProject,
	}
	if err := store.PutSource(ctx, src); err != nil {
		t.Fatal(err)
	}
	at := time.Unix(200, 0).UTC()
	if err := store.TombstoneSource(ctx, "p1", "s1", at); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetSource(ctx, "p1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsTombstoned() || got.TombstonedAt == nil || !got.TombstonedAt.Equal(at) {
		t.Fatalf("tombstone=%#v", got)
	}
	if err := store.TombstoneSource(ctx, "p1", "s1", at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	again, err := store.GetSource(ctx, "p1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if !again.TombstonedAt.Equal(at) {
		t.Fatalf("idempotent tombstone must keep first time: %#v", again.TombstonedAt)
	}
	src.TombstonedAt = nil
	if err := store.PutSource(ctx, src); err != nil {
		t.Fatal(err)
	}
	revived, err := store.GetSource(ctx, "p1", "s1")
	if err != nil || revived.IsTombstoned() {
		t.Fatalf("put without tombstone must revive: %#v err=%v", revived, err)
	}
}

func projectIDs(projects []corpus.Project) []ids.ProjectID {
	out := make([]ids.ProjectID, len(projects))
	for i, p := range projects {
		out[i] = p.ID
	}
	return out
}
