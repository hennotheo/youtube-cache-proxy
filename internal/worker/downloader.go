package worker

import (
	"context"

	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// Downloader retrieves one source and publishes its audio file locally.
type Downloader interface {
	Download(context.Context, string, string, youtube.DownloadLimits) (youtube.VideoMetadata, error)
}
