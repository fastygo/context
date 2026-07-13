package isolation_test

import (
	"testing"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/policy/isolation"
)

func TestRequireProjectMatch(t *testing.T) {
	if err := isolation.RequireProjectMatch("p1", ""); err != nil {
		t.Fatal(err)
	}
	if err := isolation.RequireProjectMatch("p1", "p1"); err != nil {
		t.Fatal(err)
	}
	err := isolation.RequireProjectMatch("p1", "p2")
	if !apperr.Is(err, apperr.Permission) {
		t.Fatalf("want permission, got %v", err)
	}
	if err := isolation.RequireProjectMatch("", "p1"); !apperr.Is(err, apperr.Validation) {
		t.Fatalf("empty bound: %v", err)
	}
}

func TestRequireTenantMatch(t *testing.T) {
	if err := isolation.RequireTenantMatch("", "t1"); err != nil {
		t.Fatal(err)
	}
	if err := isolation.RequireTenantMatch("t1", ""); err != nil {
		t.Fatal(err)
	}
	if err := isolation.RequireTenantMatch("t1", "t1"); err != nil {
		t.Fatal(err)
	}
	err := isolation.RequireTenantMatch("t1", "t2")
	if !apperr.Is(err, apperr.Permission) {
		t.Fatalf("want permission, got %v", err)
	}
}
