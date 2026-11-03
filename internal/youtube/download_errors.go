package youtube

import "errors"

var (
	// ErrVideoTooLong indicates that the video's metadata exceeds the configured duration limit.
	ErrVideoTooLong = errors.New("video exceeds maximum duration")
	// ErrDownloadTooLarge indicates that metadata or downloaded bytes exceed the configured size limit.
	ErrDownloadTooLarge = errors.New("download exceeds maximum size")
	// ErrVideoDurationUnknown indicates that yt-dlp did not report a finite video duration.
	ErrVideoDurationUnknown = errors.New("video duration is unavailable")
)
