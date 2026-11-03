package store

import (
	"context"
	"fmt"
)

// CreateSource adds one video in pending state, rejecting duplicate video IDs.
func (s *Store) CreateSource(ctx context.Context, videoID, url string) (Source, error) {
	if videoID == "" || url == "" {
		return Source{}, fmt.Errorf("video ID and URL are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback()
	source, err := scanSource(tx.QueryRowContext(ctx, `INSERT INTO sources
		(video_id, url, status, created_at, updated_at)
		VALUES (?, ?, 'pending', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		RETURNING `+sourceColumns, videoID, url))
	if err != nil {
		if isUniqueConstraint(err) {
			return Source{}, ErrDuplicate
		}
		return Source{}, fmt.Errorf("create source: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Source{}, fmt.Errorf("commit source: %w", err)
	}
	return source, nil
}
