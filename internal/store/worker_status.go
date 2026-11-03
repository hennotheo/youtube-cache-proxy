package store

import "context"

// WorkerStatus contains persisted queue counts and the active manual batch.
type WorkerStatus struct {
	Paused                bool
	Pending               int
	Downloading           int
	Ready                 int
	Failed                int
	SuccessfulDownloads24 int
	ActiveRun             *DownloadRun
}

// GetWorkerStatus returns queue counts, pause state, and the current active batch.
func (s *Store) GetWorkerStatus(ctx context.Context) (WorkerStatus, error) {
	var status WorkerStatus
	var paused int
	err := s.db.QueryRowContext(ctx, `SELECT is_paused FROM worker_state WHERE id = 1`).Scan(&paused)
	if err != nil {
		return WorkerStatus{}, err
	}
	status.Paused = paused != 0
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM sources GROUP BY status`)
	if err != nil {
		return WorkerStatus{}, err
	}
	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return WorkerStatus{}, err
		}
		switch state {
		case "pending":
			status.Pending = count
		case "downloading":
			status.Downloading = count
		case "ready":
			status.Ready = count
		case "failed":
			status.Failed = count
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return WorkerStatus{}, err
	}
	if err := rows.Close(); err != nil {
		return WorkerStatus{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM download_attempts
		WHERE status = 'succeeded' AND finished_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-24 hours')`).
		Scan(&status.SuccessfulDownloads24); err != nil {
		return WorkerStatus{}, err
	}
	run, err := s.GetActiveRun(ctx)
	if err == nil {
		status.ActiveRun = &run
	} else if err != ErrNotFound {
		return WorkerStatus{}, err
	}
	return status, nil
}
