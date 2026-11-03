package store

import (
	"context"
	"fmt"
)

// ListSources returns a page of newest sources, optionally filtered by status and video ID.
func (s *Store) ListSources(ctx context.Context, status, videoID string, limit, offset int) ([]Source, error) {
	if status != "" && status != "pending" && status != "downloading" && status != "ready" && status != "failed" {
		return nil, fmt.Errorf("invalid source status")
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+sourceColumns+` FROM sources
		WHERE (? = '' OR status = ?) AND (? = '' OR video_id = ?)
		ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, status, status, videoID, videoID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()
	var sources []Source
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}
