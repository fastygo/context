package devcli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/storage/memory"
)

func TestExportAndDeleteProject(t *testing.T) {
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")

	root := t.TempDir()
	corpusDir := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	arch := filepath.Join(root, "project.archive.json")
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpusDir, "note.md"), []byte("# Note\n\nDELETEME42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpusDir, "proj_del", "Del"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_del", ""); err != nil {
		t.Fatal(err)
	}
	exp, err := devcli.ExportProject(data, "proj_del", arch)
	if err != nil || !exp.OK || exp.Chunks < 1 {
		t.Fatalf("export=%#v err=%v", exp, err)
	}
	if _, err := os.Stat(arch); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.DeleteProject(data, "proj_del", "wrong"); !apperr.Is(err, apperr.Validation) {
		t.Fatalf("want confirm validation, got %v", err)
	}
	del, err := devcli.DeleteProject(data, "proj_del", "proj_del")
	if err != nil || !del.WorkspaceCleared {
		t.Fatalf("delete=%#v err=%v", del, err)
	}
	if _, err := (devcli.Workspace{DataDir: data}).Load(); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("workspace must be gone: %v", err)
	}
	if _, err := devcli.Search(data, "proj_del", "DELETEME42", "exact", ""); err == nil {
		t.Fatal("search after delete must fail")
	}
}

func TestMemoryDeleteProjectPurgesRows(t *testing.T) {
	t.Parallel()
	store := memory.New()
	ctx := context.Background()
	if err := store.PutProject(ctx, corpus.Project{ID: "p1", Name: "A"}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutSource(ctx, corpus.Source{
		ID: "s1", ProjectID: "p1", Type: corpus.SourceTypeFile,
		PathKey: "a", TrustLevel: foundation.TrustProject,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteProject(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetProject(ctx, "p1"); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("project: %v", err)
	}
	if _, err := store.GetSource(ctx, "p1", "s1"); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("source: %v", err)
	}
}
