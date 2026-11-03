package store

import (
	"context"
	"database/sql"
	"fmt"
)

// CompleteAttempt marks the attempt and its source successful in one transaction.
func (s *Store) CompleteAttempt(ctx context.Context, attemptID int64, audio AudioResult) error {
	if audio.Path == "" || audio.Mime == "" || audio.SizeBytes < 0 || audio.DurationSeconds < 0 {
		return fmt.Errorf("invalid audio result")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	sourceID, err := startedAttemptSource(ctx, tx, attemptID)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE sources SET status = 'ready',
		audio_path = ?, audio_mime = ?, audio_size_bytes = ?, error_code = NULL, error_message = NULL,
		completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ? AND status = 'downloading'`, audio.Path, audio.Mime, audio.SizeBytes, sourceID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ErrConflict
	}
	metadata := audio.Metadata
	if metadata.Title == "" {
		metadata.Title = audio.Title
	}
	if metadata.DurationSeconds <= 0 {
		metadata.DurationSeconds = audio.DurationSeconds
	}
	if err := persistSourceMetadata(ctx, tx, sourceID, metadata); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE download_attempts SET status = 'succeeded',
		finished_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, attemptID); err != nil {
		return err
	}
	return tx.Commit()
}

func startedAttemptSource(ctx context.Context, tx *sql.Tx, attemptID int64) (int64, error) {
	var sourceID int64
	var status string
	err := tx.QueryRowContext(ctx, `SELECT source_id, status FROM download_attempts WHERE id = ?`, attemptID).Scan(&sourceID, &status)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if status != "started" {
		return 0, ErrConflict
	}
	return sourceID, nil
}
