package store

import "context"

// HasPendingRunSource reports whether an active run still has a pending source.
func (s *Store) HasPendingRunSource(ctx context.Context, runID int64) (bool, error) {
	var pending int
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM download_run_sources AS rs
		JOIN download_runs AS r ON r.id = rs.run_id
		JOIN sources AS source ON source.id = rs.source_id
		WHERE rs.run_id = ? AND r.status = 'active' AND source.status = 'pending'
	)`, runID).Scan(&pending)
	return pending != 0, err
}
