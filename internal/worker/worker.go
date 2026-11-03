package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// Worker reserves and processes manual batches in the background.
type Worker struct {
	ctx        context.Context
	queue      Queue
	downloader Downloader
	audioDir   string
	limits     youtube.DownloadLimits
	runs       sync.WaitGroup
}

// New creates a worker tied to the service context.
func New(ctx context.Context, queue Queue, downloader Downloader, audioDir string, limits youtube.DownloadLimits) (*Worker, error) {
	if ctx == nil {
		return nil, fmt.Errorf("worker context is required")
	}
	if queue == nil || downloader == nil {
		return nil, fmt.Errorf("worker queue and downloader are required")
	}
	if audioDir == "" {
		return nil, fmt.Errorf("worker audio directory is required")
	}
	return &Worker{ctx: ctx, queue: queue, downloader: downloader, audioDir: audioDir, limits: limits}, nil
}

// Start atomically reserves a batch and returns without waiting for downloads.
func (w *Worker) Start(ctx context.Context, count int) (store.DownloadRun, error) {
	if ctx == nil {
		return store.DownloadRun{}, fmt.Errorf("request context is required")
	}
	run, err := w.queue.ReserveRun(ctx, count)
	if err != nil {
		return store.DownloadRun{}, err
	}
	w.runs.Add(1)
	go w.process(run)
	return run, nil
}

// Wait waits for all batches started by this worker to finish.
func (w *Worker) Wait() {
	w.runs.Wait()
}

// process claims and handles every source reserved for a run serially.
func (w *Worker) process(run store.DownloadRun) {
	defer w.runs.Done()
	failed := false
	for {
		source, attempt, err := w.queue.StartNext(w.ctx, run.ID)
		if errors.Is(err, store.ErrNoPending) {
			break
		}
		if errors.Is(err, store.ErrPaused) || w.ctx.Err() != nil {
			return
		}
		if err != nil {
			w.closeRun(run.ID, "partial", "worker could not claim the next source")
			return
		}
		attemptFailed, err := w.processAttempt(source, attempt)
		if err != nil {
			if w.ctx.Err() != nil {
				return
			}
			w.closeRun(run.ID, "partial", "worker could not record an attempt")
			return
		}
		failed = failed || attemptFailed
	}
	status, reason := "completed", ""
	if failed {
		status, reason = "partial", "one or more downloads failed"
	}
	w.closeRun(run.ID, status, reason)
}

// processAttempt downloads one reserved source and persists its result.
func (w *Worker) processAttempt(source store.Source, attempt store.DownloadAttempt) (bool, error) {
	template := filepath.Join(w.audioDir, source.VideoID+".%(ext)s")
	metadata, err := w.downloader.Download(w.ctx, source.VideoID, template, w.limits)
	if err != nil {
		return true, w.recordFailure(attempt.ID, downloadFailureCode(err), err.Error())
	}
	audio, err := audioResult(w.audioDir, source.VideoID, metadata)
	if err != nil {
		return true, w.recordFailure(attempt.ID, "invalid_output", err.Error())
	}
	if err := w.queue.CompleteAttempt(w.ctx, attempt.ID, audio); err != nil {
		return false, err
	}
	return false, nil
}

// recordFailure stores an attempt failure without scheduling an in-run retry.
func (w *Worker) recordFailure(attemptID int64, code, message string) error {
	return w.queue.FailAttempt(w.ctx, attemptID, code, message, false)
}

// closeRun stores a terminal status for a batch that can no longer continue.
func (w *Worker) closeRun(runID int64, status, reason string) {
	if err := w.queue.CloseRun(w.ctx, runID, status, reason); err != nil && w.ctx.Err() == nil {
		log.Printf("close download run %d: %v", runID, err)
	}
}

// downloadFailureCode maps yt-dlp classification to a stable persisted code.
func downloadFailureCode(err error) string {
	if youtube.IsRetryableDownloadError(err) {
		return "temporary_download_failure"
	}
	return "permanent_download_failure"
}
