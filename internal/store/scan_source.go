package store

import "database/sql"

const sourceColumns = `id, video_id, url, status, title, duration_seconds, audio_path,
	audio_mime, audio_size_bytes, error_code, error_message, attempt_count,
	created_at, updated_at, started_at, completed_at, channel_name, channel_id, channel_url,
	thumbnail_url, published_at, description, tags_json, categories_json, view_count, language, live_status`

func scanSource(row interface{ Scan(...any) error }) (Source, error) {
	var source Source
	err := row.Scan(&source.ID, &source.VideoID, &source.URL, &source.Status,
		&source.Title, &source.DurationSeconds, &source.AudioPath, &source.AudioMime,
		&source.AudioSizeBytes, &source.ErrorCode, &source.ErrorMessage, &source.AttemptCount,
		&source.CreatedAt, &source.UpdatedAt, &source.StartedAt, &source.CompletedAt,
		&source.ChannelName, &source.ChannelID, &source.ChannelURL, &source.ThumbnailURL,
		&source.PublishedAt, &source.Description, &source.TagsJSON, &source.CategoriesJSON,
		&source.ViewCount, &source.Language, &source.LiveStatus)
	if err == sql.ErrNoRows {
		return Source{}, ErrNotFound
	}
	return source, err
}
