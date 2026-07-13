package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/httpserver"
)

// TestLabGateSmoke is the Chunk 32 offline Lab gate regression path.
func TestLabGateSmoke(t *testing.T) {
	t.Setenv("CONTEXT_COMPLETER_KIND", "localecho")
	t.Setenv("CONTEXT_ENABLE_DENSE", "")
	t.Setenv("CONTEXT_FAIL_METADATA", "")
	t.Setenv("CONTEXT_FAIL_VECTOR", "")
	t.Setenv("CONTEXT_REDACT", "1")

	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join(corpus, "notes.md")
	if err := os.WriteFile(doc, []byte("# Gate\n\nExact token ZEBRA42 for Lab smoke.\napi_key=notforlabpreview99\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(dataDir, corpus, "proj_lab", "Lab Gate"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(dataDir, "proj_lab", ""); err != nil {
		t.Fatal(err)
	}

	srv, err := httpserver.New(httpserver.Config{DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	assertNoHostPath := func(t *testing.T, raw []byte) {
		t.Helper()
		s := string(raw)
		if strings.Contains(s, `E:\`) || strings.Contains(s, `E:/`) {
			t.Fatalf("host path leaked: %s", s)
		}
		// Temp dirs on Windows often start with the drive; also reject Unix abs roots in JSON values.
		if strings.Contains(s, `"corpus_root"`) || strings.Contains(s, dataDir) || strings.Contains(s, corpus) {
			t.Fatalf("workspace path leaked: %s", s)
		}
	}

	get := func(path string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		return rr
	}
	post := func(path string, body any) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw)))
		return rr
	}

	rr := get("/health")
	if rr.Code != http.StatusOK {
		t.Fatalf("health: %d %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("X-Context-API-Version") != "v1" {
		t.Fatalf("version header: %q", rr.Header().Get("X-Context-API-Version"))
	}
	var health map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &health)
	if health["api_version"] != "v1" || health["ok"] != true {
		t.Fatalf("health body: %#v", health)
	}
	assertNoHostPath(t, rr.Body.Bytes())

	rr = get("/v1/ready?project_id=proj_lab")
	if rr.Code != http.StatusOK {
		t.Fatalf("ready: %d %s", rr.Code, rr.Body.String())
	}

	rr = get("/v1/status?project_id=proj_lab")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d %s", rr.Code, rr.Body.String())
	}
	assertNoHostPath(t, rr.Body.Bytes())

	rr = post("/v1/search", map[string]string{"project_id": "proj_lab", "query": "ZEBRA42", "mode": "hybrid"})
	if rr.Code != http.StatusOK {
		t.Fatalf("search: %d %s", rr.Code, rr.Body.String())
	}
	assertNoHostPath(t, rr.Body.Bytes())

	rr = post("/v1/context-pack", map[string]string{"project_id": "proj_lab", "query": "ZEBRA42"})
	if rr.Code != http.StatusOK {
		t.Fatalf("pack: %d %s", rr.Code, rr.Body.String())
	}
	assertNoHostPath(t, rr.Body.Bytes())

	rr = post("/v1/inspect", map[string]string{"project_id": "proj_lab", "query": "ZEBRA42"})
	if rr.Code != http.StatusOK {
		t.Fatalf("inspect: %d %s", rr.Code, rr.Body.String())
	}
	assertNoHostPath(t, rr.Body.Bytes())
	if strings.Contains(rr.Body.String(), "notforlabpreview99") {
		t.Fatal("inspect leaked secret preview")
	}

	rr = get("/v1/metrics?project_id=proj_lab")
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics: %d %s", rr.Code, rr.Body.String())
	}
	assertNoHostPath(t, rr.Body.Bytes())

	rr = get("/v1/quota?project_id=proj_lab")
	if rr.Code != http.StatusOK {
		t.Fatalf("quota: %d %s", rr.Code, rr.Body.String())
	}

	rr = post("/v1/agent-run", map[string]string{"project_id": "proj_lab", "query": "ZEBRA42"})
	if rr.Code != http.StatusOK {
		t.Fatalf("agent: %d %s", rr.Code, rr.Body.String())
	}
	assertNoHostPath(t, rr.Body.Bytes())
	var agent map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &agent)
	if text, _ := agent["model_text"].(string); strings.Contains(text, "notforlabpreview99") {
		t.Fatalf("agent leaked secret: %q", text)
	}

	rr = post("/v1/jobs", map[string]string{
		"project_id": "proj_lab", "query": "ZEBRA42", "owner": "lab-gate",
	})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("job start: %d %s", rr.Code, rr.Body.String())
	}
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	job, _ := start["job"].(map[string]any)
	jobID, _ := job["id"].(string)
	if jobID == "" || job["owner"] != "lab-gate" {
		t.Fatalf("%#v", start)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		rr = get("/v1/jobs/" + jobID + "?project_id=proj_lab")
		if rr.Code != http.StatusOK {
			t.Fatalf("job status: %d %s", rr.Code, rr.Body.String())
		}
		assertNoHostPath(t, rr.Body.Bytes())
		var st map[string]any
		_ = json.Unmarshal(rr.Body.Bytes(), &st)
		if st["status"] == "completed" {
			return
		}
		if st["status"] == "failed" {
			t.Fatalf("job failed: %#v", st)
		}
		if time.Now().After(deadline) {
			t.Fatalf("job timeout: %#v", st)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
