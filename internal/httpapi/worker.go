package httpapi

import "net/http"

type workerStateResponse struct {
	Paused                bool           `json:"paused"`
	Pending               int            `json:"pending"`
	Downloading           int            `json:"downloading"`
	Ready                 int            `json:"ready"`
	Failed                int            `json:"failed"`
	SuccessfulDownloads24 int            `json:"successful_downloads_24h"`
	ActiveRun             *activeRunView `json:"active_run,omitempty"`
}

type activeRunView struct {
	ID             int64  `json:"id"`
	RequestedCount int    `json:"requested_count"`
	ReservedCount  int    `json:"reserved_count"`
	Status         string `json:"status"`
}

func (a *API) pauseWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := a.worker.Pause(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not pause worker")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"paused": true})
}

func (a *API) resumeWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := a.worker.Resume(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not resume worker")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"paused": false})
}

func (a *API) workerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	status, err := a.repository.GetWorkerStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read worker status")
		return
	}
	response := workerStateResponse{
		Paused: status.Paused, Pending: status.Pending, Downloading: status.Downloading,
		Ready: status.Ready, Failed: status.Failed, SuccessfulDownloads24: status.SuccessfulDownloads24,
	}
	if status.ActiveRun != nil {
		response.ActiveRun = &activeRunView{
			ID: status.ActiveRun.ID, RequestedCount: status.ActiveRun.RequestedCount,
			ReservedCount: status.ActiveRun.ReservedCount, Status: status.ActiveRun.Status,
		}
	}
	writeJSON(w, http.StatusOK, response)
}
