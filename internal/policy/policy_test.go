package policy_test

import (
	"testing"

	"github.com/fastygo/context/internal/policy"
)

func TestDecisionValidation(t *testing.T) {
	t.Parallel()
	if err := policy.DecisionAllow.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := policy.Decision("maybe").Validate(); err == nil {
		t.Fatal("unknown decision should fail")
	}
}

func TestPolicySnapshotRejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := (policy.PolicySnapshot{}).Validate(); err == nil {
		t.Fatal("expected zero policy snapshot to fail")
	}
}
