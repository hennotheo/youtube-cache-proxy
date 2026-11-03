package store

import "context"

// RecoverInterrupted closes active runs and resets unfinished downloads after startup.
func (s *Store) RecoverInterrupted(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	queries := []string{
		`UPDATE download_attempts SET status = 'failed', error_code = 'interrupted',
			error_message = 'service stopped', finished_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
			WHERE status = 'started'`,
		`UPDATE sources SET status = 'pending', updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
			WHERE status = 'downloading'`,
		`UPDATE download_runs SET status = 'interrupted', stop_reason = 'service stopped',
			completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE status = 'active'`,
	}
	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return tx.Commit()
}
