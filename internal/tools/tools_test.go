package tools_test

import (
	"testing"

	"github.com/fastygo/context/internal/policy"
	"github.com/fastygo/context/internal/tools"
)

func TestToolSchemaRejectsEmptyName(t *testing.T) {
	t.Parallel()
	s := tools.ToolSchema{InputSchemaVer: "1", OutputSchemaVer: "1"}
	if err := s.Validate(); err == nil {
		t.Fatal("expected empty name to fail")
	}
}

func TestToolCallRequiresDecision(t *testing.T) {
	t.Parallel()
	c := tools.ToolCall{
		ID:        "tc1",
		ProjectID: "p1",
		ToolName:  "read_file",
		Decision:  policy.DecisionDeny,
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
