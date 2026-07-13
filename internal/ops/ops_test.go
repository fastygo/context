package ops_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fastygo/context/internal/ops"
)

func TestAppendAndListEvalHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops", "eval_history.jsonl")
	r1 := ops.EvalRecord{
		RecordedAt: time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC),
		SuiteID:    "eval-golden-v1", OK: true, Passed: 7, Failed: 0, Total: 7,
		DurationMS: 12, Summary: "first",
	}
	r2 := ops.EvalRecord{
		RecordedAt: time.Date(2026, 7, 13, 11, 0, 0, 0, time.UTC),
		SuiteID:    "eval-golden-v1", OK: false, Passed: 6, Failed: 1, Total: 7,
		DurationMS: 15, Summary: "second",
	}
	if err := ops.AppendEval(path, r1); err != nil {
		t.Fatal(err)
	}
	if err := ops.AppendEval(path, r2); err != nil {
		t.Fatal(err)
	}
	all, err := ops.ListEval(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2, got %d", len(all))
	}
	if all[0].Summary != "second" || all[1].Summary != "first" {
		t.Fatalf("newest-first: %#v", all)
	}
	lim, err := ops.ListEval(path, 1)
	if err != nil || len(lim) != 1 || lim[0].Summary != "second" {
		t.Fatalf("limit: %#v %v", lim, err)
	}
	missing, err := ops.ListEval(filepath.Join(dir, "missing.jsonl"), 10)
	if err != nil || missing != nil {
		t.Fatalf("missing file: %#v %v", missing, err)
	}
}
