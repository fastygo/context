package memory_test

import (
	"testing"

	"github.com/fastygo/context/internal/policy"
	toolfake "github.com/fastygo/context/internal/tools/fake"
	"github.com/fastygo/context/internal/tools/memory"
)

func TestRegistryRegisterAndList(t *testing.T) {
	t.Parallel()
	reg := memory.NewRegistry()
	if err := reg.Register(toolfake.ReadSnippetSchema()); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(toolfake.ReadSnippetSchema()); err == nil {
		t.Fatal("expected conflict on duplicate register")
	}
	got, ok := reg.Get(toolfake.ReadSnippetName)
	if !ok || got.RiskLevel != policy.RiskLow {
		t.Fatalf("got=%#v ok=%v", got, ok)
	}
	if len(reg.List()) != 1 {
		t.Fatal("expected one tool")
	}
}
