package models_test

import (
	"testing"

	"github.com/fastygo/context/internal/models"
)

func TestModelCallRejectsZeroValue(t *testing.T) {
	t.Parallel()
	if err := (models.ModelCall{}).Validate(); err == nil {
		t.Fatal("expected zero model call to fail")
	}
}

func TestModelCallAcceptsMinimal(t *testing.T) {
	t.Parallel()
	m := models.ModelCall{
		ID:           "mc1",
		ProjectID:    "p1",
		ProviderID:   "fake",
		ModelVersion: "fake-v1",
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}
