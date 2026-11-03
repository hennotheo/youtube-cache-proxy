package worker

import (
	"context"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
)

// Queue contains the persistent operations required to run one batch.
type Queue interface {
	// ReserveRun atomically selects and reserves a manual batch.
	ReserveRun(context.Context, int) (store.DownloadRun, error)
	// HasPendingRunSource reports whether the batch has another pending source.
	HasPendingRunSource(context.Context, int64) (bool, error)
	// HasAttemptCapacity reports whether the rolling quota permits another attempt.
	HasAttemptCapacity(context.Context, int) (bool, error)
	// StartNext claims a source unless paused or over the attempt quota.
	StartNext(context.Context, int64, int) (store.Source, store.DownloadAttempt, error)
	// CompleteAttempt persists a successful audio result.
	CompleteAttempt(context.Context, int64, store.AudioResult) error
	// FailAttempt records a failed attempt and optionally returns it to pending.
	FailAttempt(context.Context, int64, string, string, bool) error
	// CloseRun marks the batch with a terminal status.
	CloseRun(context.Context, int64, string, string) error
}
