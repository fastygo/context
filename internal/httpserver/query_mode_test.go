package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/httpserver"
)

func setupRussianHTTPWorkspace(t *testing.T) string {
	t.Helper()
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_SPARSE_KIND", "")
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"rail.md": "Вдоль железной дороги шли люди.\n",
		"road.md": "Новая дорога построена за год.\n",
		"chat.md": "чат о дороге и погоде.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(corpus, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := devcli.InitProject(data, corpus, "proj_qhttp", "Query HTTP"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_qhttp", ""); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSearchQueryModeOverHTTP(t *testing.T) {
	data := setupRussianHTTPWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: data})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	body, _ := json.Marshal(map[string]string{
		"project_id": "proj_qhttp",
		"query":      "дорога -чат",
		"mode":       "query",
		"lang":       "ru",
	})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/search", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("query search: %d %s", rr.Code, rr.Body.String())
	}
	var res struct {
		Mode       string `json:"mode"`
		Candidates []struct {
			ChunkID string `json:"chunk_id"`
		} `json:"candidates"`
		QueryExplain struct {
			Canonical string `json:"canonical"`
			Leaves    []struct {
				Kind       string   `json:"kind"`
				Expansions []string `json:"expansions"`
			} `json:"leaves"`
		} `json:"query_explain"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Mode != "query" {
		t.Fatalf("mode=%q", res.Mode)
	}
	// Two road chunks (nominative + genitive via morphology), chat excluded.
	if len(res.Candidates) != 2 {
		t.Fatalf("want 2 candidates, got %d: %s", len(res.Candidates), rr.Body.String())
	}
	if res.QueryExplain.Canonical != "(AND дорога (NOT чат))" {
		t.Fatalf("canonical=%q", res.QueryExplain.Canonical)
	}
	if len(res.QueryExplain.Leaves) == 0 || len(res.QueryExplain.Leaves[0].Expansions) == 0 {
		t.Fatalf("explain leaves missing expansions: %s", rr.Body.String())
	}
}

func TestSearchQueryModeInvalidReturns400(t *testing.T) {
	data := setupRussianHTTPWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: data})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"project_id": "proj_qhttp",
		"query":      `"незакрытая фраза`,
		"mode":       "query",
	})
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/search", bytes.NewReader(body)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid operator query must be 400, got %d %s", rr.Code, rr.Body.String())
	}
}
