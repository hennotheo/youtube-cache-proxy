package youtube

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrVideoTooLong indicates that the video's metadata exceeds the configured duration limit.
	ErrVideoTooLong = errors.New("video exceeds maximum duration")
	// ErrDownloadTooLarge indicates that metadata or downloaded bytes exceed the configured size limit.
	ErrDownloadTooLarge = errors.New("download exceeds maximum size")
	// ErrVideoDurationUnknown indicates that yt-dlp did not report a finite video duration.
	ErrVideoDurationUnknown = errors.New("video duration is unavailable")
)

var retryableHTTPStatus = regexp.MustCompile(`(?i)\bhttp error\s+(408|425|429|5[0-9]{2})\b`)

// classifyYTDLPFailure maps command diagnostics to a stable retry decision.
func classifyYTDLPFailure(stderr string, cause error) error {
	message := strings.TrimSpace(stderr)
	if message == "" && cause != nil {
		message = cause.Error()
	}
	if message == "" {
		message = "command failed without diagnostic output"
	}

	lowerMessage := strings.ToLower(message)
	retryable := retryableHTTPStatus.MatchString(message)
	for _, marker := range []string{
		"timed out",
		"timeout",
		"connection reset",
		"connection refused",
		"network is unreachable",
		"temporary failure",
		"temporary name resolution",
		"try again",
	} {
		if strings.Contains(lowerMessage, marker) {
			retryable = true
			break
		}
	}

	return &ytDLPFailure{retryable: retryable, message: message, cause: cause}
}
