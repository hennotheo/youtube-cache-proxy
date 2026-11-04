package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// Repository contains the persistent operations exposed through the REST API.
type Repository interface {
	CreateSource(context.Context, string, string) (store.Source, error)
	ListSources(context.Context, string, string, int, int) ([]store.Source, error)
	GetSource(context.Context, int64) (store.Source, error)
	DeleteSource(context.Context, int64) error
	RetrySource(context.Context, int64) error
	GetWorkerStatus(context.Context) (store.WorkerStatus, error)
	UpdateSourceMetadata(context.Context, int64, youtube.VideoMetadata) error
}

// MetadataProvider retrieves YouTube details without downloading media.
type MetadataProvider interface {
	FetchMetadata(context.Context, string) (youtube.VideoMetadata, error)
}

// BatchWorker starts and controls manual download batches.
type BatchWorker interface {
	Start(context.Context, int) (store.DownloadRun, error)
	Pause(context.Context) error
	Resume(context.Context) error
}

// API serves queue, audio, and worker operations over HTTP.
type API struct {
	repository        Repository
	worker            BatchWorker
	metadataProvider  MetadataProvider
	producerTokenHash [sha256.Size]byte
	adminTokenHash    [sha256.Size]byte
	sessionSecret     []byte
}

// NewAPI constructs the REST API with its persistent store and authentication secrets.
func NewAPI(repository Repository, worker BatchWorker, metadataProvider MetadataProvider, producerToken, adminToken, sessionSecret string) *API {
	return &API{
		repository: repository, worker: worker, metadataProvider: metadataProvider,
		producerTokenHash: sha256.Sum256([]byte(producerToken)),
		adminTokenHash:    sha256.Sum256([]byte(adminToken)),
		sessionSecret:     []byte(sessionSecret),
	}
}

// isAdminToken compares a submitted token without exposing its value.
func (a *API) isAdminToken(token string) bool {
	providedHash := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(providedHash[:], a.adminTokenHash[:]) == 1
}

func (a *API) register(mux *http.ServeMux) {
	mux.HandleFunc("/session/login", a.loginSession)
	mux.Handle("/session/csrf", a.requireRole(adminRole, a.csrfToken))
	mux.Handle("/session/logout", a.requireRole(adminRole, a.logoutSession))
	mux.Handle("/sources", a.requireRole(producerRole, a.sources))
	mux.Handle("/sources/", a.requireRoleFunc(a.sourceRole, a.source))
	mux.Handle("/downloads/run", a.requireRole(adminRole, a.runDownloads))
	mux.Handle("/worker/pause", a.requireRole(adminRole, a.pauseWorker))
	mux.Handle("/worker/resume", a.requireRole(adminRole, a.resumeWorker))
	mux.Handle("/worker/status", a.requireRole(adminRole, a.workerStatus))
}

