package youtube

// VideoMetadata contains the yt-dlp metadata retained for a completed download.
type VideoMetadata struct {
	// Title is the title returned by yt-dlp.
	Title string
	// DurationSeconds is the rounded-up media duration.
	DurationSeconds int64
	// SizeBytes is the measured size of the completed output file.
	SizeBytes int64
	// ChannelName is the channel name, falling back to the uploader name.
	ChannelName string
	// ChannelID is the channel identifier, falling back to the uploader identifier.
	ChannelID string
	// ChannelURL is the channel URL, falling back to the uploader URL.
	ChannelURL string
	// ThumbnailURL is the best thumbnail URL supplied by yt-dlp.
	ThumbnailURL string
	// PublishedAt is the publication date formatted as YYYY-MM-DD.
	PublishedAt string
	// Description is the video description.
	Description string
	// Tags contains the video tags supplied by yt-dlp.
	Tags []string
	// Categories contains the video categories supplied by yt-dlp.
	Categories []string
	// ViewCount is the view count when yt-dlp supplies it.
	ViewCount *int64
	// Language is the video's language when yt-dlp supplies it.
	Language string
	// LiveStatus describes the video's live-stream state when available.
	LiveStatus string
}
