package worker

import (
	"context"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
)

// Queue contains the persistent operations required to run one batch.
type Queue interface {
	ReserveRun(context.Context, int) (store.DownloadRun, error)
	StartNext(context.Context, int64) (store.Source, store.DownloadAttempt, error)
	CompleteAttempt(context.Context, int64, store.AudioResult) error
	FailAttempt(context.Context, int64, string, string, bool) error
	CloseRun(context.Context, int64, string, string) error
}
