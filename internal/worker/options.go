package worker

import (
	"time"

	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// Options contains the limits and pacing settings for a batch worker.
type Options struct {
	// AudioDir is the directory where completed audio files are published.
	AudioDir string
	// Limits bounds the duration, size, rate, and runtime of a download.
	Limits youtube.DownloadLimits
	// MaxAttemptsPer24Hours is the maximum number of starts in the rolling window.
	MaxAttemptsPer24Hours int
	// PauseMin is the shortest delay between attempts.
	PauseMin time.Duration
	// PauseMax is the longest delay between attempts.
	PauseMax time.Duration
}
