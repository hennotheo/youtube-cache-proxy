package store

import (
	"context"
	"database/sql"
	"fmt"
)

func insertRun(ctx context.Context, tx *sql.Tx, count int, ids []int64, videos []string) (DownloadRun, error) {
	var run DownloadRun
	err := tx.QueryRowContext(ctx, `INSERT INTO download_runs
		(requested_count, reserved_count, status, created_at)
		VALUES (?, ?, 'active', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		RETURNING id, requested_count, reserved_count, status, created_at`, count, len(ids)).
		Scan(&run.ID, &run.RequestedCount, &run.ReservedCount, &run.Status, &run.CreatedAt)
	if err != nil {
		if isUniqueConstraint(err) {
			return DownloadRun{}, ErrConflict
		}
		return DownloadRun{}, fmt.Errorf("insert run: %w", err)
	}
	for position, sourceID := range ids {
		if _, err := tx.ExecContext(ctx, `INSERT INTO download_run_sources
			(run_id, source_id, video_id, position) VALUES (?, ?, ?, ?)`, run.ID, sourceID, videos[position], position+1); err != nil {
			return DownloadRun{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return DownloadRun{}, err
	}
	return run, nil
}
