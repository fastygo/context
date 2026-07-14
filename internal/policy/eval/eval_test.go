package eval_test

import (
	"testing"

	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/policy/eval"
	toolfake "github.com/fastygo/context/internal/tools/fake"
)

func TestWriteRequiresAskWithoutRule(t *testing.T) {
	t.Parallel()
	eng := eval.Engine{
		Snapshot: policy.PolicySnapshot{ID: "pol1", ProjectID: "p1", Version: "v1"},
		Default:  policy.DecisionAllow, // must not silently allow writes
	}
	d, err := eng.Decide(toolfake.WriteNoteName, toolfake.WriteNoteSchema())
	if err != nil {
		t.Fatal(err)
	}
	if d != policy.DecisionAsk {
		t.Fatalf("want ask for write side-effect, got %s", d)
	}
}

func TestReadDefaultsDenyWithoutRule(t *testing.T) {
	t.Parallel()
	eng := eval.Engine{
		Snapshot: policy.PolicySnapshot{ID: "pol1", ProjectID: "p1", Version: "v1"},
	}
	d, err := eng.Decide(toolfake.ReadSnippetName, toolfake.ReadSnippetSchema())
	if err != nil {
		t.Fatal(err)
	}
	if d != policy.DecisionDeny {
		t.Fatalf("want deny for unread rule, got %s", d)
	}
}

func TestExplicitAllowWriteStillAllows(t *testing.T) {
	t.Parallel()
	eng := eval.Engine{
		Snapshot: policy.PolicySnapshot{
			ID: "pol1", ProjectID: "p1", Version: "v1",
			Rules: []policy.Rule{{
				Name: "allow-write", ToolName: toolfake.WriteNoteName, Decision: policy.DecisionAllow,
			}},
		},
	}
	d, err := eng.Decide(toolfake.WriteNoteName, toolfake.WriteNoteSchema())
	if err != nil {
		t.Fatal(err)
	}
	if d != policy.DecisionAllow {
		t.Fatalf("want allow, got %s", d)
	}
}
