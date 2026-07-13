package devcli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/devcli"
)

func TestSearchDegradedWhenDenseFailInjected(t *testing.T) {
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.md"), []byte("# Alpha\n\nZEBRA42 token\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_deg", "Degraded"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_deg", ""); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONTEXT_ENABLE_DENSE", "1")
	t.Setenv("CONTEXT_FAIL_VECTOR", "1")

	res, err := devcli.Search(data, "proj_deg", "ZEBRA42", "hybrid", "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Degraded || len(res.DegradedReasons) == 0 {
		t.Fatalf("want degraded: %#v", res)
	}
	if len(res.Candidates) == 0 {
		t.Fatal("exact/sparse should still return hits")
	}

	_, err = devcli.Search(data, "proj_deg", "ZEBRA42", "dense", "")
	if err == nil || !apperr.Is(err, apperr.Unavailable) {
		t.Fatalf("dense mode want unavailable, got %v", err)
	}
}

func TestReadyFailInject(t *testing.T) {
	t.Setenv("CONTEXT_FAIL_EMBEDDER", "1")
	rep, err := devcli.Ready(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Ready || !rep.Degraded {
		t.Fatalf("%#v", rep)
	}
}
