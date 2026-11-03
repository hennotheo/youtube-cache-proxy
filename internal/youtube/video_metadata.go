package youtube

// VideoMetadata contains the yt-dlp metadata retained for a completed download.
type VideoMetadata struct {
	// Title is the title returned by yt-dlp.
	Title string
	// DurationSeconds is the rounded-up media duration.
	DurationSeconds int64
	// SizeBytes is the measured size of the completed output file.
	SizeBytes int64
}
