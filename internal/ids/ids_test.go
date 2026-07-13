package ids_test

import (
	"testing"

	"github.com/fastygo/context/internal/ids"
)

func TestProjectIDRejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := ids.ProjectID("").Validate(); err == nil {
		t.Fatal("expected empty project_id to fail validation")
	}
}

func TestProjectIDAcceptsNonEmpty(t *testing.T) {
	t.Parallel()
	if err := ids.ProjectID("proj_1").Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
