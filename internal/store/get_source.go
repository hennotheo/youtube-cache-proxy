package store

import "context"

// GetSource returns a source by its database identifier.
func (s *Store) GetSource(ctx context.Context, id int64) (Source, error) {
	return scanSource(s.db.QueryRowContext(ctx, `SELECT `+sourceColumns+` FROM sources WHERE id = ?`, id))
}
