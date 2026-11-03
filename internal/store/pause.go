package store

import "context"

// SetPaused persists whether new attempts may start.
func (s *Store) SetPaused(ctx context.Context, paused bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE worker_state SET is_paused = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = 1`, paused)
	return err
}
