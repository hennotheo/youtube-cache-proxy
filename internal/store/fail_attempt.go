package store

import (
	"context"
	"fmt"
)

// FailAttempt records an error and returns the source to pending only for an in-run retry.
func (s *Store) FailAttempt(ctx context.Context, attemptID int64, code, message string, retry bool) error {
	if code == "" {
		return fmt.Errorf("error code is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	sourceID, err := startedAttemptSource(ctx, tx, attemptID)
	if err != nil {
		return err
	}
	nextStatus := "failed"
	if retry {
		nextStatus = "pending"
	}
	result, err := tx.ExecContext(ctx, `UPDATE sources SET status = ?, error_code = ?, error_message = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ? AND status = 'downloading'`,
		nextStatus, code, message, sourceID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ErrConflict
	}
	if _, err := tx.ExecContext(ctx, `UPDATE download_attempts SET status = 'failed', error_code = ?,
		error_message = ?, finished_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
		code, message, attemptID); err != nil {
		return err
	}
	return tx.Commit()
}
