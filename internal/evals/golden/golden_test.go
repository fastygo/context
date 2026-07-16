package golden_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fastygo/context/internal/evals/golden"
)

func TestGoldenSuiteOffline(t *testing.T) {
	rep, err := golden.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK {
		raw, _ := json.MarshalIndent(rep, "", "  ")
		t.Fatalf("golden suite failed:\n%s", raw)
	}
	if len(rep.Cases) != len(golden.Specs()) {
		t.Fatalf("cases=%d", len(rep.Cases))
	}
	kinds := map[string]bool{}
	for _, c := range rep.Cases {
		kinds[c.Kind] = true
		if !c.Passed {
			t.Fatalf("case %s failed: %#v", c.ID, c)
		}
	}
	for _, k := range []string{"exact", "sparse", "dense", "hybrid", "multilingual", "lexicon", "pack_verify", "morph_ru", "query"} {
		if !kinds[k] {
			t.Fatalf("missing kind %s", k)
		}
	}
}

func TestReportJSONShape(t *testing.T) {
	t.Parallel()
	rep, err := golden.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := golden.MarshalReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ok", "suite_id", "generated_at", "cases", "summary"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %s", key)
		}
	}
}
