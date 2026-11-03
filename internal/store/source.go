package store

import "database/sql"

// Source is one queued YouTube video and its persisted download metadata.
type Source struct {
	ID              int64
	VideoID         string
	URL             string
	Status          string
	Title           sql.NullString
	DurationSeconds sql.NullInt64
	AudioPath       sql.NullString
	AudioMime       sql.NullString
	AudioSizeBytes  sql.NullInt64
	ErrorCode       sql.NullString
	ErrorMessage    sql.NullString
	AttemptCount    int
	CreatedAt       string
	UpdatedAt       string
	StartedAt       sql.NullString
	CompletedAt     sql.NullString
	// ChannelName is the channel or uploader name returned by yt-dlp.
	ChannelName sql.NullString
	// ChannelID is the channel or uploader identifier returned by yt-dlp.
	ChannelID sql.NullString
	// ChannelURL is the channel or uploader URL returned by yt-dlp.
	ChannelURL sql.NullString
	// ThumbnailURL is the remote video thumbnail URL.
	ThumbnailURL sql.NullString
	// PublishedAt is the source publication date in YYYY-MM-DD format.
	PublishedAt sql.NullString
	// Description is the video description returned by yt-dlp.
	Description sql.NullString
	// TagsJSON stores video tags as a JSON array.
	TagsJSON sql.NullString
	// CategoriesJSON stores video categories as a JSON array.
	CategoriesJSON sql.NullString
	// ViewCount is the number of views when available.
	ViewCount sql.NullInt64
	// Language is the video language when available.
	Language sql.NullString
	// LiveStatus records the live-stream state when available.
	LiveStatus sql.NullString
}
