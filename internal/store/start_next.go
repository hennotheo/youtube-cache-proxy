package store

import (
	"context"
	"database/sql"
	"fmt"
)

// StartNext atomically claims the next reserved source within the daily attempt quota.
func (s *Store) StartNext(ctx context.Context, runID int64, maxAttemptsPer24Hours int) (Source, DownloadAttempt, error) {
	if maxAttemptsPer24Hours < 1 {
		return Source{}, DownloadAttempt{}, fmt.Errorf("daily attempt quota must be positive")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Source{}, DownloadAttempt{}, err
	}
	defer tx.Rollback()
	if err := checkCanStart(ctx, tx, runID); err != nil {
		return Source{}, DownloadAttempt{}, err
	}
	attempt, err := claimNext(ctx, tx, runID)
	if err != nil {
		return Source{}, DownloadAttempt{}, err
	}
	if err := checkAttemptQuota(ctx, tx, maxAttemptsPer24Hours); err != nil {
		return Source{}, DownloadAttempt{}, err
	}
	source, err := scanSource(tx.QueryRowContext(ctx, `SELECT `+sourceColumns+` FROM sources WHERE id = ?`, attempt.SourceID))
	if err != nil {
		return Source{}, DownloadAttempt{}, err
	}
	if err := tx.Commit(); err != nil {
		return Source{}, DownloadAttempt{}, err
	}
	return source, attempt, nil
}

// checkAttemptQuota counts the attempt just inserted in tx and rolls it back if over limit.
func checkAttemptQuota(ctx context.Context, tx *sql.Tx, maxAttemptsPer24Hours int) error {
	attempts, err := recentAttemptCount(ctx, tx)
	if err != nil {
		return err
	}
	if attempts > maxAttemptsPer24Hours {
		return ErrQuotaExceeded
	}
	return nil
}

func checkCanStart(ctx context.Context, tx *sql.Tx, runID int64) error {
	var paused int
	if err := tx.QueryRowContext(ctx, `SELECT is_paused FROM worker_state WHERE id = 1`).Scan(&paused); err != nil {
		return err
	}
	if paused != 0 {
		return ErrPaused
	}
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM download_runs WHERE id = ?`, runID).Scan(&status); err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if status != "active" {
		return ErrConflict
	}
	var downloading int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources WHERE status = 'downloading'`).Scan(&downloading); err != nil {
		return err
	}
	if downloading != 0 {
		return ErrConflict
	}
	return nil
}
