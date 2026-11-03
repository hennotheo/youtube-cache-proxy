package store

import (
	"context"
	"fmt"

	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// CreateSource adds one video in pending state, rejecting duplicate video IDs.
func (s *Store) CreateSource(ctx context.Context, videoID, rawURL string) (Source, error) {
	if videoID == "" {
		return Source{}, fmt.Errorf("video ID is required")
	}
	parsedVideoID, normalizedURL, err := youtube.ParseVideoURL(rawURL)
	if err != nil {
		return Source{}, err
	}
	if parsedVideoID != videoID {
		return Source{}, fmt.Errorf("%w: video ID does not match URL", youtube.ErrInvalidVideoURL)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback()
	source, err := scanSource(tx.QueryRowContext(ctx, `INSERT INTO sources
		(video_id, url, status, created_at, updated_at)
		VALUES (?, ?, 'pending', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		RETURNING `+sourceColumns, videoID, normalizedURL))
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
