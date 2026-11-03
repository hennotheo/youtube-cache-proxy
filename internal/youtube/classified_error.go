package youtube

import (
	"context"
	"errors"
	"fmt"
)

// ytDLPFailure records whether a failed yt-dlp command may succeed on retry.
type ytDLPFailure struct {
	retryable bool
	message   string
	cause     error
}

func (e *ytDLPFailure) Error() string {
	classification := "permanent"
	if e.retryable {
		classification = "temporary"
	}
	return fmt.Sprintf("yt-dlp %s failure: %s", classification, e.message)
}

func (e *ytDLPFailure) Unwrap() error {
	return e.cause
}

// IsRetryableDownloadError reports whether a download failure may succeed on retry.
func IsRetryableDownloadError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var failure *ytDLPFailure
	return errors.As(err, &failure) && failure.retryable
}
