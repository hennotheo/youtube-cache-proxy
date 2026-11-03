package store

import (
	"context"
	"database/sql"
	"fmt"
)

// DeleteSource removes an unreserved source while preserving run and attempt history.
func (s *Store) DeleteSource(ctx context.Context, sourceID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin source deletion: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM sources
		WHERE id = ? AND status <> 'downloading'
		AND NOT EXISTS (
			SELECT 1 FROM download_run_sources AS rs
			JOIN download_runs AS r ON r.id = rs.run_id
			WHERE rs.source_id = sources.id AND r.status = 'active'
		)`, sourceID)
	if err != nil {
		return fmt.Errorf("delete source: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("count deleted sources: %w", err)
	} else if count == 1 {
		return tx.Commit()
	}
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM sources WHERE id = ?`, sourceID).Scan(&exists); err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("check source after deletion: %w", err)
	}
	return ErrConflict
}
