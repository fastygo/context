package apperr_test

import (
	"errors"
	"testing"

	"github.com/fastygo/context/internal/apperr"
)

func TestIsMatchesCode(t *testing.T) {
	t.Parallel()
	err := apperr.New(apperr.NotFound, "project missing")
	if !apperr.Is(err, apperr.NotFound) {
		t.Fatal("expected not_found")
	}
	if apperr.Is(err, apperr.Conflict) {
		t.Fatal("did not expect conflict")
	}
	if !errors.As(err, new(*apperr.Error)) {
		t.Fatal("expected apperr.Error")
	}
}
