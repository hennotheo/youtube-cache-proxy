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
}
