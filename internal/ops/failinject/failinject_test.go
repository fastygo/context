package failinject_test

import (
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ops/failinject"
)

func TestCheckInjected(t *testing.T) {
	t.Setenv("CONTEXT_FAIL_VECTOR", "1")
	if err := failinject.Check(failinject.Vector); err == nil || !apperr.Is(err, apperr.Unavailable) {
		t.Fatalf("want unavailable, got %v", err)
	}
	t.Setenv("CONTEXT_FAIL_VECTOR", "")
	if err := failinject.Check(failinject.Vector); err != nil {
		t.Fatal(err)
	}
}
