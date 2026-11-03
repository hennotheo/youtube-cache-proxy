package store

import (
	"context"
	"database/sql"
)

// RetrySource returns a failed source to pending without starting a download.
func (s *Store) RetrySource(ctx context.Context, sourceID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM download_run_sources AS rs
		JOIN download_runs AS r ON r.id = rs.run_id
		WHERE rs.source_id = ? AND r.status = 'active'`, sourceID).Scan(&active); err != nil {
		return err
	}
	if active != 0 {
		return ErrConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE sources SET status = 'pending', error_code = NULL,
		error_message = NULL, completed_at = NULL, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ? AND status = 'failed'`, sourceID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		var existing int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM sources WHERE id = ?`, sourceID).Scan(&existing); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		return ErrConflict
	}
	return tx.Commit()
}
