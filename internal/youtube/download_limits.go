package youtube

import (
	"fmt"
	"time"
)

// DownloadLimits bounds media length, downloaded bytes, and wall-clock time.
type DownloadLimits struct {
	// MaxBytes is the maximum completed audio size in bytes.
	MaxBytes int64
	// MaxDurationSeconds is the maximum media duration in seconds.
	MaxDurationSeconds int64
	// Timeout bounds metadata lookup and download time together.
	Timeout time.Duration
}

func (limits DownloadLimits) validate() error {
	if limits.MaxBytes <= 0 {
		return fmt.Errorf("maximum download size must be positive")
	}
	if limits.MaxDurationSeconds <= 0 {
		return fmt.Errorf("maximum video duration must be positive")
	}
	if limits.Timeout <= 0 {
		return fmt.Errorf("download timeout must be positive")
	}
	return nil
}
