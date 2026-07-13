package postgres

import (
	"context"
	"embed"
	"sort"
	"strings"

	"github.com/fastygo/context/internal/apperr"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate applies pending SQL migrations in lexicographic order.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
)`); err != nil {
		return apperr.Wrap(apperr.Internal, "schema_migrations", err)
	}
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "read migrations", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, name).Scan(&exists); err != nil {
			return apperr.Wrap(apperr.Internal, "check migration", err)
		}
		if exists {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return apperr.Wrap(apperr.Internal, "read migration "+name, err)
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return apperr.Wrap(apperr.Internal, "begin migration", err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return apperr.Wrap(apperr.Internal, "apply migration "+name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return apperr.Wrap(apperr.Internal, "record migration "+name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return apperr.Wrap(apperr.Internal, "commit migration "+name, err)
		}
	}
	return nil
}
