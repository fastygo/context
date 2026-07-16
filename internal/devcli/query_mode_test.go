package devcli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/devcli"
)

func setupRussianWorkspace(t *testing.T) string {
	t.Helper()
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	t.Setenv("CONTEXT_LANG", "")
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"rail.md":  "Вдоль железной дороги шли люди к станции.\n",
		"road.md":  "Новая дорога построена за один год.\n",
		"cozy.md":  "Домашний уют не заменит путешествий.\n",
		"house.md": "Старый дом стоит у самой реки.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(corpus, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := devcli.InitProject(data, corpus, "proj_ru", "RU"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_ru", ""); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSearchQueryModeRussianMorphology(t *testing.T) {
	data := setupRussianWorkspace(t)

	// Citation form must reach the inflected "дороги" via morph expansion.
	res, err := devcli.SearchWithLang(data, "proj_ru", "дорога", "query", "", "ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) < 2 {
		t.Fatalf("expected both road chunks, got %d: %+v", len(res.Candidates), res.Candidates)
	}
	if res.QueryExplain == nil || len(res.QueryExplain.Leaves) != 1 {
		t.Fatalf("query explain missing: %#v", res.QueryExplain)
	}
	if len(res.QueryExplain.Leaves[0].Expansions) == 0 {
		t.Fatalf("expected morph expansions in explain: %#v", res.QueryExplain.Leaves[0])
	}
}

func TestSearchQueryModeMorphPhrase(t *testing.T) {
	data := setupRussianWorkspace(t)

	res, err := devcli.SearchWithLang(data, "proj_ru", `~"железная дорога"`, "query", "", "ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("morph phrase must match exactly one chunk, got %d", len(res.Candidates))
	}
	if res.Candidates[0].Snippet == nil {
		t.Fatal("morph phrase candidate must carry a snippet")
	}
}

func TestSearchQueryModeTokenBoundaryAndNot(t *testing.T) {
	data := setupRussianWorkspace(t)

	// "дом" must match the noun chunk but not "Домашний" (substring trap),
	// and NOT must exclude the river chunk.
	res, err := devcli.SearchWithLang(data, "proj_ru", "lang:ru дом -реки", "query", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("дом chunk mentions реки; NOT must exclude it, got %d", len(res.Candidates))
	}

	res2, err := devcli.SearchWithLang(data, "proj_ru", "lang:ru дом", "query", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res2.Candidates) != 1 {
		t.Fatalf("token-boundary дом must match exactly one chunk, got %d", len(res2.Candidates))
	}
}

func TestSearchQueryModeInvalidQuery(t *testing.T) {
	data := setupRussianWorkspace(t)
	if _, err := devcli.SearchWithLang(data, "proj_ru", `"незакрытая`, "query", "", "ru"); err == nil {
		t.Fatal("invalid operator query must fail with validation error")
	}
}

func TestHybridModeUsesLangExpansion(t *testing.T) {
	data := setupRussianWorkspace(t)

	// Plain hybrid mode with CONTEXT_LANG=ru should also see morph recall.
	t.Setenv("CONTEXT_LANG", "ru")
	res, err := devcli.Search(data, "proj_ru", "дорога", "hybrid", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) < 2 {
		t.Fatalf("hybrid with ru expansion should reach inflected chunk, got %d", len(res.Candidates))
	}
}
