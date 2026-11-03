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
	result, err := s.db.ExecContext(ctx, `INSERT INTO sources (video_id, url, status, created_at, updated_at)
		VALUES (?, ?, 'pending', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`, videoID, url)
	if err != nil {
		var exists int
		if lookupErr := s.db.QueryRowContext(ctx, `SELECT 1 FROM sources WHERE video_id = ?`, videoID).Scan(&exists); lookupErr == nil {
			return Source{}, ErrDuplicate
		}
		return Source{}, fmt.Errorf("create source: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Source{}, fmt.Errorf("read source ID: %w", err)
	}
	return s.GetSource(ctx, id)
}
