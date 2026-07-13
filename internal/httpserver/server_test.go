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

func setupWorkspace(t *testing.T) (dataDir string) {
	t.Helper()
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	dataDir = filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join(corpus, "notes.md")
	if err := os.WriteFile(doc, []byte("# Alpha\n\nExact token ZEBRA42 lives here for search.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(dataDir, corpus, "proj_http", "HTTP Test"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(dataDir, "proj_http", ""); err != nil {
		t.Fatal(err)
	}
	return dataDir
}

func TestHealthAndSearchPackTrace(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/status?project_id=proj_http", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d %s", rr.Code, rr.Body.String())
	}
	var st map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st["ok"] != true {
		t.Fatalf("status ok: %#v", st)
	}
	if _, has := st["corpus_root"]; has {
		t.Fatalf("status must not leak corpus_root: %#v", st)
	}
	if st["chunks"].(float64) < 1 {
		t.Fatalf("expected chunks: %#v", st)
	}

	body, _ := json.Marshal(map[string]string{
		"project_id": "proj_http",
		"query":      "ZEBRA42",
		"mode":       "exact",
	})
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/search", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("search: %d %s", rr.Code, rr.Body.String())
	}
	var search map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &search); err != nil {
		t.Fatal(err)
	}
	cands, _ := search["candidates"].([]any)
	if len(cands) < 1 {
		t.Fatalf("expected candidates: %s", rr.Body.String())
	}

	packBody, _ := json.Marshal(map[string]string{
		"project_id": "proj_http",
		"query":      "ZEBRA42",
	})
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/context-pack", bytes.NewReader(packBody)))
	if rr.Code != http.StatusOK {
		t.Fatalf("pack: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/agent-run", bytes.NewReader(packBody)))
	if rr.Code != http.StatusOK {
		t.Fatalf("agent: %d %s", rr.Code, rr.Body.String())
	}
	var agent map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &agent); err != nil {
		t.Fatal(err)
	}
	run, _ := agent["run"].(map[string]any)
	runID, _ := run["id"].(string)
	if runID == "" {
		t.Fatalf("missing run id: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/trace?project_id=proj_http&run_id="+runID, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("trace: %d %s", rr.Code, rr.Body.String())
	}
	var trace map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &trace); err != nil {
		t.Fatal(err)
	}
	events, _ := trace["events"].([]any)
	if len(events) < 1 {
		t.Fatalf("expected trace events: %s", rr.Body.String())
	}
}

func TestAuthToken(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health should be open: %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/status", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("bearer: %d %s", rr.Code, rr.Body.String())
	}
}

func TestIngestRejectsAbsolutePathKey(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"project_id": "proj_http",
		"path_key":   filepath.Join(dataDir, "escape.md"),
	})
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/ingest", bytes.NewReader(body)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestEvalOffline(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/eval", bytes.NewReader([]byte("{}"))))
	if rr.Code != http.StatusOK && rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("eval: %d %s", rr.Code, rr.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	report, _ := res["report"].(map[string]any)
	if report == nil {
		t.Fatalf("missing report: %s", rr.Body.String())
	}
}
