package evals_test

import (
	"testing"

	"github.com/fastygo/context/internal/evals"
)

func TestEvaluationRejectsZeroValue(t *testing.T) {
	t.Parallel()
	if err := (evals.Evaluation{}).Validate(); err == nil {
		t.Fatal("expected zero evaluation to fail")
	}
}
