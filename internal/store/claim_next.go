package store

import (
	"context"
	"database/sql"
)

func claimNext(ctx context.Context, tx *sql.Tx, runID int64) (DownloadAttempt, error) {
	var sourceID int64
	var videoID string
	err := tx.QueryRowContext(ctx, `SELECT s.id, s.video_id FROM download_run_sources AS rs
		JOIN sources AS s ON s.id = rs.source_id
		WHERE rs.run_id = ? AND s.status = 'pending' ORDER BY rs.position LIMIT 1`, runID).Scan(&sourceID, &videoID)
	if err == sql.ErrNoRows {
		return DownloadAttempt{}, ErrNoPending
	}
	if err != nil {
		return DownloadAttempt{}, err
	}
	var number int
	err = tx.QueryRowContext(ctx, `UPDATE sources SET status = 'downloading', attempt_count = attempt_count + 1,
		started_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ? AND status = 'pending' RETURNING attempt_count`, sourceID).Scan(&number)
	if err == sql.ErrNoRows {
		return DownloadAttempt{}, ErrConflict
	}
	if err != nil {
		return DownloadAttempt{}, err
	}
	var attempt DownloadAttempt
	err = tx.QueryRowContext(ctx, `INSERT INTO download_attempts
		(run_id, source_id, video_id, attempt_number, status, started_at)
		VALUES (?, ?, ?, ?, 'started', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		RETURNING id, run_id, source_id, video_id, attempt_number, status, started_at`,
		runID, sourceID, videoID, number).Scan(&attempt.ID, &attempt.RunID, &attempt.SourceID,
		&attempt.VideoID, &attempt.AttemptNumber, &attempt.Status, &attempt.StartedAt)
	return attempt, err
}
