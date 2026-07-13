package devcli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/config"
	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/storage/postgres"
)

func TestIngestPinsVersionsOffline(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "memory")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")

	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Note\n\nPinned versions matter.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, root, "pins", "Pins"); err != nil {
		t.Fatal(err)
	}
	st, err := devcli.Ingest(data, "pins", root)
	if err != nil {
		t.Fatal(err)
	}
	if st.Snapshot.DenseEnabled {
		t.Fatal("dense should be off by default")
	}
	if st.Snapshot.EmbedModelVersion != config.DefaultEmbeddingVersion {
		t.Fatalf("embed=%q", st.Snapshot.EmbedModelVersion)
	}
	if len(st.Chunks) == 0 {
		t.Fatal("expected chunks")
	}
	ch := st.Chunks[0]
	if ch.ChunkerVersion == "" || ch.EmbeddingVersion == "" || ch.MorphVersion == "" {
		t.Fatalf("pins missing: %#v", ch)
	}
	if ch.EmbeddingVersion != config.DefaultEmbeddingVersion {
		t.Fatalf("chunk emb=%q", ch.EmbeddingVersion)
	}
	if ch.SparseVersion != "fake" {
		t.Fatalf("sparse=%q", ch.SparseVersion)
	}
}

func TestDenseCommitOnIngestIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run dense commit integration")
	}
	t.Setenv("CONTEXT_PG_DSN", dsn)
	t.Setenv("CONTEXT_ENABLE_DENSE", "1")
	t.Setenv("CONTEXT_METADATA_KIND", "postgres")
	t.Setenv("CONTEXT_DENSE_REBUILD", "")

	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Note\n\nContextPack dense commit on ingest.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, root, "dense-commit", "DenseCommit"); err != nil {
		t.Fatal(err)
	}
	st, err := devcli.Ingest(data, "dense-commit", root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if !st.Snapshot.DenseEnabled || st.Snapshot.Status != foundation.SnapshotReady {
		t.Fatalf("snap=%#v", st.Snapshot)
	}
	if st.Chunks[0].EmbeddingVersion == "" || st.Chunks[0].ChunkerVersion == "" {
		t.Fatalf("chunk pins=%#v", st.Chunks[0])
	}

	// Search dense without rebuild — vectors must already exist from ingest.
	t.Setenv("CONTEXT_DENSE_REBUILD", "0")
	res, err := devcli.Search(data, "dense-commit", "ContextPack dense", "dense", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Backend != "postgres_pgvector" || len(res.Candidates) == 0 {
		t.Fatalf("search=%#v", res)
	}

	store, err := postgres.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.GetChunk(context.Background(), st.Project.ID, st.Chunks[0].ChunkID)
	if err != nil {
		t.Fatal(err)
	}
	if got.EmbeddingVersion != st.Chunks[0].EmbeddingVersion || got.MorphVersion != st.Chunks[0].MorphVersion {
		t.Fatalf("meta chunk=%#v want pins from %#v", got, st.Chunks[0])
	}
	snap, err := store.GetSnapshot(context.Background(), st.Project.ID, st.Snapshot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.DenseEnabled || snap.EmbedModelVersion == "" {
		t.Fatalf("meta snap=%#v", snap)
	}
}
