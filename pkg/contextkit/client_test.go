package contextkit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fastygo/context/pkg/contextkit"
)

func TestClientAgainstMockServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(contextkit.APIVersionHeader, contextkit.APIVersion)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "service": "context-serve", "api_version": contextkit.APIVersion,
		})
	})
	mux.HandleFunc("GET /v1/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "permission", "message": "unauthorized"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "project_id": "p1", "chunks": 2,
		})
	})
	mux.HandleFunc("POST /v1/search", func(w http.ResponseWriter, r *http.Request) {
		var req contextkit.SearchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"project_id": req.ProjectID,
			"snapshot_id": "snap_1",
			"query": req.Query,
			"mode": "exact",
			"candidates": []map[string]any{{"chunk_id": "c1", "merged_score": 1.0}},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cli := &contextkit.Client{BaseURL: srv.URL, Token: "secret"}
	ctx := context.Background()

	h, err := cli.Health(ctx)
	if err != nil || !h.OK {
		t.Fatalf("health: %+v %v", h, err)
	}
	if h.APIVersion != contextkit.APIVersion {
		t.Fatalf("api_version body: %q", h.APIVersion)
	}
	if cli.LastAPIVersion != contextkit.APIVersion {
		t.Fatalf("LastAPIVersion: %q", cli.LastAPIVersion)
	}
	st, err := cli.Status(ctx, "p1")
	if err != nil || st.Chunks != 2 {
		t.Fatalf("status: %+v %v", st, err)
	}
	sr, err := cli.Search(ctx, contextkit.SearchRequest{ProjectID: "p1", Query: "ZEBRA"})
	if err != nil || len(sr.Candidates) != 1 || sr.Candidates[0].ChunkID != "c1" {
		t.Fatalf("search: %+v %v", sr, err)
	}

	cli.Token = "wrong"
	_, err = cli.Status(ctx, "p1")
	if err == nil {
		t.Fatal("expected auth error")
	}
	ae, ok := err.(contextkit.APIError)
	if !ok || ae.Code != "permission" {
		t.Fatalf("want APIError permission, got %#v", err)
	}
}
