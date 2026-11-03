package store

import (
	"context"
	"database/sql"
	"fmt"
)

func applyMigration(ctx context.Context, db *sql.DB, name, script string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start migration %s: %w", name, err)
	}
	defer tx.Rollback()
	var applied int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %s: %w", name, err)
	}
	if applied != 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, script); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}
