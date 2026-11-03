package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// UpdateSourceMetadata stores optional YouTube details without changing the source state.
func (s *Store) UpdateSourceMetadata(ctx context.Context, sourceID int64, metadata youtube.VideoMetadata) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin source metadata update: %w", err)
	}
	defer tx.Rollback()
	if err := persistSourceMetadata(ctx, tx, sourceID, metadata); err != nil {
		return err
	}
	return tx.Commit()
}

type metadataExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func persistSourceMetadata(ctx context.Context, execer metadataExecer, sourceID int64, metadata youtube.VideoMetadata) error {
	tags, err := encodeStringList(metadata.Tags)
	if err != nil {
		return fmt.Errorf("encode source tags: %w", err)
	}
	categories, err := encodeStringList(metadata.Categories)
	if err != nil {
		return fmt.Errorf("encode source categories: %w", err)
	}
	result, err := execer.ExecContext(ctx, `UPDATE sources SET title = COALESCE(?, title), duration_seconds = COALESCE(?, duration_seconds),
		channel_name = COALESCE(?, channel_name), channel_id = COALESCE(?, channel_id), channel_url = COALESCE(?, channel_url),
		thumbnail_url = COALESCE(?, thumbnail_url), published_at = COALESCE(?, published_at),
		description = COALESCE(?, description), tags_json = COALESCE(?, tags_json), categories_json = COALESCE(?, categories_json),
		view_count = COALESCE(?, view_count), language = COALESCE(?, language), live_status = COALESCE(?, live_status),
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
		optionalText(metadata.Title), optionalDuration(metadata.DurationSeconds),
		optionalText(metadata.ChannelName), optionalText(metadata.ChannelID), optionalText(metadata.ChannelURL),
		optionalText(metadata.ThumbnailURL), optionalText(metadata.PublishedAt), optionalText(metadata.Description),
		tags, categories, optionalViewCount(metadata.ViewCount), optionalText(metadata.Language),
		optionalText(metadata.LiveStatus), sourceID)
	if err != nil {
		return fmt.Errorf("update source metadata: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count source metadata update: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func encodeStringList(values []string) (any, error) {
	if values == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(values)
	return string(encoded), err
}

func optionalText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func optionalDuration(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func optionalViewCount(value *int64) any {
	if value == nil || *value < 0 {
		return nil
	}
	return *value
}
