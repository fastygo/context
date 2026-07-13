// Package isolation enforces project-scoped access checks (ADR-0025).
// Authentication and quota enforcement are deferred; this package only
// rejects cross-project widening.
package isolation

import (
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
)

// RequireProjectMatch rejects a non-empty request project_id that differs from
// the bound workspace project. Empty request means "use bound project".
func RequireProjectMatch(bound, requested ids.ProjectID) error {
	if err := bound.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "bound project_id", err)
	}
	if requested == "" {
		return nil
	}
	if requested != bound {
		return apperr.New(apperr.Permission, "project id mismatch")
	}
	return nil
}

// SameTenant reports whether both tenant ids are empty (single-tenant) or equal.
func SameTenant(a, b ids.TenantID) bool {
	return a == b
}

// RequireTenantMatch rejects a non-empty request tenant that differs from bound.
// Empty request or empty bound (local single-tenant) is allowed.
func RequireTenantMatch(bound, requested ids.TenantID) error {
	if requested == "" || bound == "" {
		return nil
	}
	if requested != bound {
		return apperr.New(apperr.Permission, "tenant id mismatch")
	}
	return nil
}
