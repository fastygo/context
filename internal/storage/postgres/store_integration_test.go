package postgres_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/storage/postgres"
	"github.com/fastygo/context/internal/tracing"
)

func integrationDSN(t *testing.T) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run postgres metadata integration tests")
	}
	return dsn
}

func TestMetadataSurvivesRestart(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	projectID := ids.ProjectID("meta-int-" + time.Now().UTC().Format("150405000"))

	store1, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	temporal := &corpus.TemporalMetadata{
		Range: corpus.TemporalRange{
			Start: now.Add(-time.Hour),
			End:   now,
			Basis: corpus.TimeBasisOccurred,
		},
		IngestedAt: now,
	}
	if err := store1.PutProject(ctx, corpus.Project{ID: projectID, Name: "integration"}); err != nil {
		t.Fatal(err)
	}
	if err := store1.PutSource(ctx, corpus.Source{
		ID: "s1", ProjectID: projectID, Type: corpus.SourceTypeFile, PathKey: "readme",
		TrustLevel: foundation.TrustProject, Checksum: "aa", TemporalMetadata: temporal,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store1.PutArtifactMeta(ctx, artifacts.Artifact{
		ID: "a-in", ProjectID: projectID, MediaType: "application/json", ByteSize: 2,
		Checksum: "bb", StorageURI: "local://a-in", ArtifactType: artifacts.TypeBlob,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store1.PutArtifactMeta(ctx, artifacts.Artifact{
		ID: "a-out", ProjectID: projectID, MediaType: "application/json", ByteSize: 4,
		Checksum: "cc", StorageURI: "local://a-out",
		ArtifactType: artifacts.TypeStructured, SchemaID: "uxspec.screen.v1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store1.PutArtifactLineage(ctx, artifacts.ArtifactLineage{
		ProjectID: projectID, OutputArtifactID: "a-out", InputArtifactIDs: []ids.ArtifactID{"a-in"},
		GeneratorID: "test-gen", GeneratorVersion: "v1", TransformationKind: "derive",
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store1.PutChunk(ctx, corpus.Chunk{
		ID: "c1", ProjectID: projectID, SourceID: "s1", ArtifactID: "a-in", SnapshotID: "snap1",
		ChunkerVersion: "para-v1", Span: foundation.ByteSpan{Start: 0, End: 4},
		TextChecksum: "dd", ChunkHash: "ee", Language: "en", TemporalMetadata: temporal,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store1.PutSnapshot(ctx, indexing.IndexSnapshot{
		ID: "snap1", ProjectID: projectID, Status: foundation.SnapshotReady,
		SourceMerkleRoot: "ff", ChunkSetHash: "gg", ParserVersion: "p1", ChunkerVersion: "para-v1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store1.SetActiveSnapshot(ctx, projectID, "snap1"); err != nil {
		t.Fatal(err)
	}
	if err := store1.AppendTrace(ctx, tracing.Event{
		ID: "e1", ProjectID: projectID, RunID: "r1", Type: tracing.EventRunStarted,
		Timestamp: now, Payload: map[string]string{"k": "v"}, AnalyzerVersion: "simple-v1",
		DictionaryVersion: "dict-v1", SnapshotID: "snap1",
	}); err != nil {
		t.Fatal(err)
	}
	sensePayload, _ := json.Marshal(map[string]string{"definition": "to jog", "lemma": "run"})
	if err := store1.PutDocument(ctx, storage.MetaDocument{
		ProjectID: projectID, Kind: storage.DocumentSense, ID: "sense-run",
		Language: "en", LexemeID: "lex-run", SenseID: "sense-run",
		AnalyzerVersion: "simple-v1", DictionaryVersion: "dict-v1",
		Payload: sensePayload,
	}); err != nil {
		t.Fatal(err)
	}
	store1.Close()

	store2, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer store2.Close()

	project, err := store2.GetProject(ctx, projectID)
	if err != nil || project.ActiveSnapshotID != "snap1" {
		t.Fatalf("project=%#v err=%v", project, err)
	}
	src, err := store2.GetSource(ctx, projectID, "s1")
	if err != nil || src.TemporalMetadata == nil || src.TemporalMetadata.Range.Basis != corpus.TimeBasisOccurred {
		t.Fatalf("source temporal=%#v err=%v", src.TemporalMetadata, err)
	}
	art, err := store2.GetArtifactMeta(ctx, projectID, "a-out")
	if err != nil || art.SchemaID != "uxspec.screen.v1" || art.ArtifactType != artifacts.TypeStructured {
		t.Fatalf("artifact=%#v err=%v", art, err)
	}
	lineage, err := store2.GetArtifactLineage(ctx, projectID, "a-out")
	if err != nil || len(lineage.InputArtifactIDs) != 1 || lineage.InputArtifactIDs[0] != "a-in" {
		t.Fatalf("lineage=%#v err=%v", lineage, err)
	}
	ch, err := store2.GetChunk(ctx, projectID, "c1")
	if err != nil || ch.Language != "en" || ch.TemporalMetadata == nil {
		t.Fatalf("chunk=%#v err=%v", ch, err)
	}
	events, err := store2.ListTrace(ctx, projectID, "r1")
	if err != nil || len(events) != 1 || events[0].AnalyzerVersion != "simple-v1" {
		t.Fatalf("trace=%#v err=%v", events, err)
	}
	doc, err := store2.GetDocument(ctx, projectID, storage.DocumentSense, "sense-run")
	if err != nil || doc.LexemeID != "lex-run" || !strings.Contains(string(doc.Payload), "jog") {
		t.Fatalf("doc=%#v err=%v", doc, err)
	}
}

func TestSetActiveSnapshotTransaction(t *testing.T) {
	dsn := integrationDSN(t)
	ctx := context.Background()
	store, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	projectID := ids.ProjectID("meta-tx-" + time.Now().UTC().Format("150405000"))
	if err := store.PutProject(ctx, corpus.Project{ID: projectID, Name: "tx"}); err != nil {
		t.Fatal(err)
	}
	ready := indexing.IndexSnapshot{
		ID: "snap-a", ProjectID: projectID, Status: foundation.SnapshotReady,
		SourceMerkleRoot: "aa", ChunkSetHash: "bb", ParserVersion: "p", ChunkerVersion: "c",
	}
	if err := store.PutSnapshot(ctx, ready); err != nil {
		t.Fatal(err)
	}
	ready.ID = "snap-b"
	if err := store.PutSnapshot(ctx, ready); err != nil {
		t.Fatal(err)
	}
	if err := store.SetActiveSnapshot(ctx, projectID, "snap-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetActiveSnapshot(ctx, projectID, "snap-b"); err != nil {
		t.Fatal(err)
	}
	prev, err := store.GetSnapshot(ctx, projectID, "snap-a")
	if err != nil || prev.Status != foundation.SnapshotSuperseded {
		t.Fatalf("prev=%#v err=%v", prev, err)
	}
	err = store.SetActiveSnapshot(ctx, projectID, "missing")
	if !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
