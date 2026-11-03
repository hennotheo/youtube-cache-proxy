package store

import (
	"context"
	"database/sql"
	"fmt"
)

// CloseRun finishes an active manual batch with a terminal status.
func (s *Store) CloseRun(ctx context.Context, runID int64, status, reason string) error {
	if status != "completed" && status != "partial" && status != "interrupted" {
		return fmt.Errorf("invalid terminal run status")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE download_runs SET status = ?, stop_reason = ?,
		completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ? AND status = 'active'`,
		status, reason, runID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 1 {
		return nil
	}
	var existing int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM download_runs WHERE id = ?`, runID).Scan(&existing)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrConflict
}
