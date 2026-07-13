package devcli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/context/internal/devcli"
	"github.com/fastygo/context/internal/ops"
)

func TestEvalHistoryAndMetrics(t *testing.T) {
	root := t.TempDir()
	corpus := filepath.Join(root, "corpus")
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "a.md"), []byte("# hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.InitProject(data, corpus, "proj_ops", "Ops"); err != nil {
		t.Fatal(err)
	}
	if _, err := devcli.Ingest(data, "proj_ops", ""); err != nil {
		t.Fatal(err)
	}

	hist := ops.HistoryPath(data)
	r1, err := devcli.RunEval("", hist)
	if err != nil {
		t.Fatal(err)
	}
	if r1.History == "" || !r1.Report.OK {
		t.Fatalf("eval1: %#v", r1)
	}
	if _, err := devcli.RunEval("", hist); err != nil {
		t.Fatal(err)
	}

	histRes, err := devcli.EvalHistory(hist, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(histRes.Records) != 2 {
		t.Fatalf("history len=%d", len(histRes.Records))
	}

	m, err := devcli.Metrics(data)
	if err != nil {
		t.Fatal(err)
	}
	if !m.OK || m.Chunks < 1 || m.EvalHistoryCount != 2 || m.LastEval == nil {
		t.Fatalf("metrics: %#v", m)
	}
	if m.EvalHistoryPathKey != ops.DefaultHistoryRel {
		t.Fatalf("path key: %s", m.EvalHistoryPathKey)
	}
}
