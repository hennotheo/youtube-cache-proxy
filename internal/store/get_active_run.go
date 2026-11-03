package store

import (
	"context"
	"database/sql"
)

// GetActiveRun returns the manual batch that is currently reserved for processing.
func (s *Store) GetActiveRun(ctx context.Context) (DownloadRun, error) {
	var run DownloadRun
	err := s.db.QueryRowContext(ctx, `SELECT id, requested_count, reserved_count, status, created_at
		FROM download_runs WHERE status = 'active' ORDER BY id DESC LIMIT 1`).
		Scan(&run.ID, &run.RequestedCount, &run.ReservedCount, &run.Status, &run.CreatedAt)
	if err == sql.ErrNoRows {
		return DownloadRun{}, ErrNotFound
	}
	return run, err
}
