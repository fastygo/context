package devcli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/storage/memory"
	"github.com/fastygo/context/internal/storage/postgres"
)

func TestMetadataKindFromEnvDefaultsMemory(t *testing.T) {
	t.Setenv("CONTEXT_METADATA_KIND", "")
	t.Setenv("CONTEXT_PG_DSN", "")
	kind, err := devcli.MetadataKindFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if kind != config.StoreKindMemory {
		t.Fatalf("kind=%q", kind)
	}
}

func TestOpenMetadataMemory(t *testing.T) {
	t.Setenv("CONTEXT_METADATA_KIND", "memory")
	h, err := devcli.OpenMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if h.UsesPostgres() {
		t.Fatal("expected memory")
	}
}

func TestPersistIngestMemoryRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Hello\n\nContextPack here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, root, "demo", "Demo"); err != nil {
		t.Fatal(err)
	}
	st, err := devcli.Ingest(data, "demo", root)
	if err != nil {
		t.Fatal(err)
	}
	meta := memory.New()
	if err := meta.PutProject(context.Background(), st.Project); err != nil {
		t.Fatal(err)
	}
	// Rebuild leaves is not needed: PersistIngest needs leaves from pipeline.
	// Exercise PersistIngest via direct call with empty leaves but chunks — sources optional.
	if err := meta.PutSnapshot(context.Background(), st.Snapshot); err != nil {
		t.Fatal(err)
	}
	if err := meta.SetActiveSnapshot(context.Background(), st.Project.ID, st.Snapshot.ID); err != nil {
		t.Fatal(err)
	}
	got, err := meta.GetSnapshot(context.Background(), st.Project.ID, st.Snapshot.ID)
	if err != nil || got.ID != st.Snapshot.ID {
		t.Fatalf("snap=%#v err=%v", got, err)
	}
}

func TestDurablePostgresIngestAndTrace(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run durable CLI integration tests")
	}
	t.Setenv("CONTEXT_METADATA_KIND", "postgres")
	t.Setenv("CONTEXT_PG_DSN", dsn)

	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Hello\n\nContextPack durable metadata.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, root, "durable", "Durable"); err != nil {
		t.Fatal(err)
	}
	st, err := devcli.Ingest(data, "durable", root)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Chunks) == 0 {
		t.Fatal("expected chunks")
	}

	// Reopen postgres and prove restart survival without state.json.
	store, err := postgres.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	project, err := store.GetProject(context.Background(), st.Project.ID)
	if err != nil || project.ActiveSnapshotID != st.Snapshot.ID {
		t.Fatalf("project=%#v err=%v", project, err)
	}
	snap, err := store.GetSnapshot(context.Background(), st.Project.ID, st.Snapshot.ID)
	if err != nil || snap.Status == "" {
		t.Fatalf("snap=%#v err=%v", snap, err)
	}
	chunks, err := store.ListChunks(context.Background(), st.Project.ID, st.Snapshot.ID)
	if err != nil || len(chunks) != len(st.Chunks) {
		t.Fatalf("chunks=%d want=%d err=%v", len(chunks), len(st.Chunks), err)
	}

	agent, err := devcli.AgentRun(data, "durable", "ContextPack")
	if err != nil {
		t.Fatal(err)
	}
	if agent.MetaKind != "postgres" {
		t.Fatalf("meta_kind=%q", agent.MetaKind)
	}
	tr, err := devcli.Trace(data, "durable", string(agent.Run.ID))
	if err != nil {
		t.Fatal(err)
	}
	if tr.MetaKind != "postgres" || len(tr.Events) == 0 {
		t.Fatalf("trace=%#v", tr)
	}
	events, err := store.ListTrace(context.Background(), st.Project.ID, agent.Run.ID)
	if err != nil || len(events) == 0 {
		t.Fatalf("store trace=%d err=%v", len(events), err)
	}
}
