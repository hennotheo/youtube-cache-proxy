package store

import "github.com/hennotheo/youtube-cache-proxy/internal/youtube"

// AudioResult contains the metadata written when a download succeeds.
type AudioResult struct {
	Title           string
	DurationSeconds int64
	Path            string
	Mime            string
	SizeBytes       int64
	// Metadata contains the full yt-dlp details retained with the completed source.
	Metadata youtube.VideoMetadata
}
