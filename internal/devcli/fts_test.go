package devcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSparseFTSIngestSearchIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run sparse FTS CLI integration")
	}
	t.Setenv("CONTEXT_SPARSE_KIND", "postgres_fts")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")

	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Note\n\nContextPack evidence and hybrid retrieval for projects.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := InitProject(data, root, "demo-fts", "Demo FTS")
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	st, err = Ingest(data, string(st.Project.ID), root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if st.Project.ActiveSnapshotID == "" {
		t.Fatal("expected snapshot")
	}

	res, err := Search(data, string(st.Project.ID), "ContextPack hybrid", "sparse", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.SparseBackend != "postgres_fts" {
		t.Fatalf("sparse_backend=%q", res.SparseBackend)
	}
	if len(res.Candidates) == 0 {
		t.Fatal("expected FTS hits")
	}
}
