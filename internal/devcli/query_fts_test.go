package devcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestQueryModeWithLiveFTSIntegration proves the operator layer over a live
// Postgres FTS sparse backend: deterministic token/morph matching decides the
// result set; FTS only reinforces scoring (ADR-0043).
func TestQueryModeWithLiveFTSIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CONTEXT_PG_DSN"))
	if dsn == "" {
		t.Skip("set CONTEXT_PG_DSN to run query-mode FTS integration")
	}
	t.Setenv("CONTEXT_SPARSE_KIND", "postgres_fts")
	t.Setenv("CONTEXT_METADATA_KIND", "memory")
	t.Setenv("CONTEXT_ENABLE_DENSE", "")

	root := t.TempDir()
	data := t.TempDir()
	files := map[string]string{
		"rail.md": "Вдоль железной дороги шли люди к станции.\n",
		"road.md": "Новая дорога построена за один год.\n",
		"chat.md": "чат о дороге и погоде.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	st, err := InitProject(data, root, "demo-qlfts", "Query FTS")
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	st, err = Ingest(data, string(st.Project.ID), root)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	res, err := SearchWithLang(data, string(st.Project.ID), "дорога -чат", "query", "", "ru")
	if err != nil {
		t.Fatalf("SearchWithLang: %v", err)
	}
	if res.SparseBackend != "postgres_fts" {
		t.Fatalf("sparse_backend=%q", res.SparseBackend)
	}
	// Morphology reaches both railway (дороги) and road chunks; chat excluded.
	if len(res.Candidates) != 2 {
		t.Fatalf("want 2 candidates, got %d: %+v", len(res.Candidates), res.Candidates)
	}
	if res.QueryExplain == nil || res.QueryExplain.Canonical != "(AND дорога (NOT чат))" {
		t.Fatalf("query explain: %#v", res.QueryExplain)
	}

	phrase, err := SearchWithLang(data, string(st.Project.ID), `~"железная дорога"`, "query", "", "ru")
	if err != nil {
		t.Fatalf("morph phrase: %v", err)
	}
	if len(phrase.Candidates) != 1 {
		t.Fatalf("morph phrase want 1 candidate, got %d", len(phrase.Candidates))
	}
	if phrase.Candidates[0].Snippet == nil {
		t.Fatal("morph phrase must carry snippet")
	}
}
