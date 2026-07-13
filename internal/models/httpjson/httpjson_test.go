package httpjson_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/models"
	"github.com/fastygo/context/internal/models/httpjson"
	"github.com/fastygo/context/internal/retrieval"
)

func TestHTTPCompleterAndEmbedder(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/complete", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text": "remote-answer", "provider_id": "demo", "model_version": "demo-v1",
		})
	})
	mux.HandleFunc("POST /v1/embed", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"vectors": [][]float32{{0.1, 0.2, 0.3}}, "model_version": "emb-v1",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	comp := httpjson.Completer{BaseURL: srv.URL}
	out, err := comp.Complete(context.Background(), models.CompletionRequest{
		ProjectID: "p1",
		Pack: retrieval.ContextPack{
			ID: "pack1", ProjectID: "p1", RetrievalPlanID: "plan1",
			Checksum: "abc",
			EvidenceItems: []retrieval.EvidenceItem{{
				ID: "e1", Class: foundation.EvidenceSourceText, TrustLevel: foundation.TrustProject,
				Surface: "hello",
			}},
		},
	})
	if err != nil || out.Text != "remote-answer" || out.ModelCall.ProviderID != "demo" {
		t.Fatalf("%#v %v", out, err)
	}

	emb := httpjson.Embedder{BaseURL: srv.URL, Dimension: 3}
	vecs, ver, err := emb.Embed(context.Background(), []string{"hi"})
	if err != nil || ver != "emb-v1" || len(vecs) != 1 || len(vecs[0]) != 3 {
		t.Fatalf("%v %s %v", vecs, ver, err)
	}
}
