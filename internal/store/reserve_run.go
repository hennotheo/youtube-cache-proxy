package store

import (
	"context"
	"database/sql"
	"fmt"
)

// ReserveRun snapshots up to count oldest pending sources for one manual batch.
func (s *Store) ReserveRun(ctx context.Context, count int) (DownloadRun, error) {
	if count < 1 {
		return DownloadRun{}, fmt.Errorf("run count must be positive")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return DownloadRun{}, err
	}
	defer tx.Rollback()
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM download_runs WHERE status = 'active'`).Scan(&active); err != nil {
		return DownloadRun{}, err
	}
	if active != 0 {
		return DownloadRun{}, ErrConflict
	}
	ids, videos, err := pendingSources(ctx, tx, count)
	if err != nil {
		return DownloadRun{}, err
	}
	if len(ids) == 0 {
		return DownloadRun{}, ErrNoPending
	}
	return insertRun(ctx, tx, count, ids, videos)
}

func pendingSources(ctx context.Context, tx *sql.Tx, count int) ([]int64, []string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, video_id FROM sources WHERE status = 'pending'
		ORDER BY created_at, id LIMIT ?`, count)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ids []int64
	var videos []string
	for rows.Next() {
		var id int64
		var video string
		if err := rows.Scan(&id, &video); err != nil {
			return nil, nil, err
		}
		ids, videos = append(ids, id), append(videos, video)
	}
	return ids, videos, rows.Err()
}
