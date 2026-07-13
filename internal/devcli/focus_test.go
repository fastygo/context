package devcli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/ignore"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/storage/postgres"
)

func TestIngestHonorsContextignore(t *testing.T) {
	root := t.TempDir()
	data := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("docs/keep.md", "# Keep\n\nContextPack keep.\n")
	mustWrite("vendor/lib/x.md", "# Vendor\n\nshould skip.\n")
	mustWrite(ignore.FileName, "secret/\n")
	mustWrite("secret/no.md", "# Secret\n")

	if _, err := devcli.InitProject(data, root, "ignore-demo", "Ignore"); err != nil {
		t.Fatal(err)
	}
	st, err := devcli.Ingest(data, "ignore-demo", root)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range st.Chunks {
		if strings.Contains(ch.RelativePath, "vendor") || strings.Contains(ch.RelativePath, "secret") {
			t.Fatalf("ignored path indexed: %#v", ch)
		}
	}
	if len(st.Chunks) == 0 {
		t.Fatal("expected kept chunks")
	}
}

func TestFocusRoundTripMemoryAndSearch(t *testing.T) {
	t.Setenv("CONTEXT_METADATA_KIND", "memory")
	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Note\n\nFocus lens ContextPack.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, root, "focus-demo", "Focus"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "focus-demo", root); err != nil {
		t.Fatal(err)
	}
	focus := retrieval.FocusProfile{
		ID:                 "focus_cli",
		Objective:          "Focus lens ContextPack",
		RequiredTrustLevel: foundation.TrustProject,
		CitationStrictness: "strict",
		ContextBudget:      retrieval.Budget{MaxItems: 3, MaxChars: 500},
	}
	put, err := devcli.PutFocus(data, "focus-demo", focus)
	if err != nil {
		t.Fatal(err)
	}
	if put.Focus.ID != "focus_cli" {
		t.Fatalf("%#v", put)
	}
	got, kind, err := devcli.GetFocus(data, "focus-demo", "focus_cli")
	if err != nil || got.ContextBudget.MaxItems != 3 {
		t.Fatalf("got=%#v kind=%s err=%v", got, kind, err)
	}
	search, err := devcli.Search(data, "focus-demo", "ContextPack", "hybrid", "focus_cli")
	if err != nil {
		t.Fatal(err)
	}
	if search.FocusID != ids.FocusID("focus_cli") {
		t.Fatalf("focus_id=%q", search.FocusID)
	}
	pack, err := devcli.BuildPack(data, "focus-demo", "ContextPack", "focus_cli")
	if err != nil {
		t.Fatal(err)
	}
	if pack.FocusID != "focus_cli" || pack.Pack.Budget.MaxItems != 3 {
		t.Fatalf("pack=%#v", pack)
	}
}

func TestFocusPostgresSurvivesRestart(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN for focus postgres integration")
	}
	t.Setenv("CONTEXT_PG_DSN", dsn)
	t.Setenv("CONTEXT_METADATA_KIND", "postgres")

	root := t.TempDir()
	data := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("# Note\n\nDurable focus.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proj := "focus-pg"
	if _, err := devcli.InitProject(data, root, proj, "FocusPG"); err != nil {
		t.Fatal(err)
	}
	focus := retrieval.FocusProfile{
		ID:                 "focus_pg",
		Objective:          "Durable focus",
		RequiredTrustLevel: foundation.TrustProject,
		ContextBudget:      retrieval.Budget{MaxItems: 2, MaxChars: 200},
	}
	if _, err := devcli.PutFocus(data, proj, focus); err != nil {
		t.Fatal(err)
	}

	store, err := postgres.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.GetFocus(context.Background(), ids.ProjectID(proj), "focus_pg")
	if err != nil || got.Objective != "Durable focus" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	list, err := store.ListFocus(context.Background(), ids.ProjectID(proj))
	if err != nil || len(list) == 0 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}
