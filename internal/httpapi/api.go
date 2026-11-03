package httpapi

import (
	"context"
	"net/http"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
)

// Repository contains the persistent operations exposed through the REST API.
type Repository interface {
	CreateSource(context.Context, string, string) (store.Source, error)
	ListSources(context.Context, string, int, int) ([]store.Source, error)
	GetSource(context.Context, int64) (store.Source, error)
	DeleteSource(context.Context, int64) error
	RetrySource(context.Context, int64) error
	GetWorkerStatus(context.Context) (store.WorkerStatus, error)
}

// BatchWorker starts and controls manual download batches.
type BatchWorker interface {
	Start(context.Context, int) (store.DownloadRun, error)
	Pause(context.Context) error
	Resume(context.Context) error
}

// API serves queue, audio, and worker operations over HTTP.
type API struct {
	repository Repository
	worker     BatchWorker
}

// NewAPI constructs the REST API from its persistent store and worker.
func NewAPI(repository Repository, worker BatchWorker) *API {
	return &API{repository: repository, worker: worker}
}

func (a *API) register(mux *http.ServeMux) {
	mux.HandleFunc("/sources", a.sources)
	mux.HandleFunc("/sources/", a.source)
	mux.HandleFunc("/downloads/run", a.runDownloads)
	mux.HandleFunc("/worker/pause", a.pauseWorker)
	mux.HandleFunc("/worker/resume", a.resumeWorker)
	mux.HandleFunc("/worker/status", a.workerStatus)
}
