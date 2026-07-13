package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	if rr.Header().Get("X-Context-API-Version") != "v1" {
		t.Fatalf("api version header: %q", rr.Header().Get("X-Context-API-Version"))
	}
	var health map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health["api_version"] != "v1" {
		t.Fatalf("health api_version: %#v", health)
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

func TestProjectMismatchForbidden(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/status?project_id=other", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d %s", rr.Code, rr.Body.String())
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

func TestInspectHTTP(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"project_id": "proj_http",
		"query":      "ZEBRA42",
	})
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/inspect", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("inspect: %d %s", rr.Code, rr.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res["ok"] != true {
		t.Fatalf("%#v", res)
	}
	selected, _ := res["selected"].([]any)
	if len(selected) < 1 {
		t.Fatalf("want selected: %#v", res)
	}
	if _, has := res["corpus_root"]; has {
		t.Fatal("must not leak corpus_root")
	}
}

func TestRepairHTTP(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{
		"project_id": "proj_http",
		"mode":       "rebuild",
		"target":     "all",
	})
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/repair", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("repair: %d %s", rr.Code, rr.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res["ok"] != true {
		t.Fatalf("%#v", res)
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
	// Second eval builds history; metrics exposes last_eval.
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/eval", bytes.NewReader([]byte("{}"))))
	if rr.Code != http.StatusOK && rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("eval2: %d %s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics: %d %s", rr.Code, rr.Body.String())
	}
	var metrics map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &metrics); err != nil {
		t.Fatal(err)
	}
	if metrics["eval_history_count"].(float64) < 2 {
		t.Fatalf("want history>=2: %#v", metrics)
	}
	if metrics["last_eval"] == nil {
		t.Fatalf("missing last_eval: %#v", metrics)
	}
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/eval/history?limit=5", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("history: %d %s", rr.Code, rr.Body.String())
	}
}

func TestQuotaEndpointAndDeny(t *testing.T) {
	dataDir := setupWorkspace(t)
	t.Setenv("CONTEXT_QUOTA_MAX_PACKS", "1")
	t.Setenv("CONTEXT_QUOTA_MAX_RUNS", "")
	t.Setenv("CONTEXT_QUOTA_MAX_CHUNKS", "")

	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/quota?project_id=proj_http", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("quota: %d %s", rr.Code, rr.Body.String())
	}
	var q map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &q); err != nil {
		t.Fatal(err)
	}
	if q["decision"] != "allow" {
		t.Fatalf("%#v", q)
	}

	body, _ := json.Marshal(map[string]string{"project_id": "proj_http", "query": "ZEBRA42"})
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/context-pack", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("pack1: %d %s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/context-pack", bytes.NewReader(body)))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 on second pack: %d %s", rr.Code, rr.Body.String())
	}
}

func TestReadyEndpoint(t *testing.T) {
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/ready", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("ready: %d %s", rr.Code, rr.Body.String())
	}

	t.Setenv("CONTEXT_FAIL_METADATA", "1")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/ready", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503: %d %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health liveness: %d", rr.Code)
	}
	var health map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health["ready"] != false {
		t.Fatalf("health ready: %#v", health)
	}
}

func TestJobStartAndStatus(t *testing.T) {
	t.Setenv("CONTEXT_COMPLETER_KIND", "localecho")
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	dataDir := setupWorkspace(t)
	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	body, _ := json.Marshal(map[string]string{
		"project_id": "proj_http", "query": "ZEBRA42", "owner": "lab",
	})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewReader(body)))
	if rr.Code != http.StatusAccepted {
		t.Fatalf("job start: %d %s", rr.Code, rr.Body.String())
	}
	var start map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &start); err != nil {
		t.Fatal(err)
	}
	job, _ := start["job"].(map[string]any)
	id, _ := job["id"].(string)
	if id == "" {
		t.Fatalf("%#v", start)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/jobs/"+id+"?project_id=proj_http", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status: %d %s", rr.Code, rr.Body.String())
		}
		var st map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &st)
		if st["status"] == "completed" {
			return
		}
		if st["status"] == "failed" {
			t.Fatalf("%#v", st)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout %#v", st)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
