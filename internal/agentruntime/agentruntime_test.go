package agentruntime_test

import (
	"testing"

	"github.com/fastygo/context/internal/agentruntime"
)

func TestAgentRunRejectsZeroValue(t *testing.T) {
	t.Parallel()
	if err := (agentruntime.AgentRun{}).Validate(); err == nil {
		t.Fatal("expected zero agent run to fail")
	}
}

func TestAgentRunAcceptsMinimal(t *testing.T) {
	t.Parallel()
	r := agentruntime.AgentRun{
		ID:        "run1",
		ProjectID: "p1",
		Mode:      agentruntime.RunModeForeground,
		Status:    agentruntime.RunStatusPending,
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}
