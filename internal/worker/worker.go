package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

// Worker reserves and processes manual batches in the background.
type Worker struct {
	ctx        context.Context
	queue      Queue
	downloader Downloader
	options    Options
	runs       sync.WaitGroup
	stateMu    sync.Mutex
	running    bool
	wait       func(context.Context, time.Duration) error
	pause      func(time.Duration, time.Duration) time.Duration
}

// New creates a worker tied to the service context.
func New(ctx context.Context, queue Queue, downloader Downloader, options Options) (*Worker, error) {
	if ctx == nil {
		return nil, fmt.Errorf("worker context is required")
	}
	if queue == nil || downloader == nil {
		return nil, fmt.Errorf("worker queue and downloader are required")
	}
	if options.AudioDir == "" {
		return nil, fmt.Errorf("worker audio directory is required")
	}
	if options.MaxAttemptsPer24Hours < 1 {
		return nil, fmt.Errorf("worker daily attempt quota must be positive")
	}
	if options.PauseMin < 0 || options.PauseMax < options.PauseMin {
		return nil, fmt.Errorf("worker pause range is invalid")
	}
	if options.PauseMin%time.Second != 0 || options.PauseMax%time.Second != 0 {
		return nil, fmt.Errorf("worker pause range must use whole seconds")
	}
	return &Worker{
		ctx: ctx, queue: queue, downloader: downloader, options: options,
		wait: waitForPause, pause: randomPause,
	}, nil
}

// Start atomically reserves a batch and returns without waiting for downloads.
func (w *Worker) Start(ctx context.Context, count int) (store.DownloadRun, error) {
	if ctx == nil {
		return store.DownloadRun{}, fmt.Errorf("request context is required")
	}
	w.stateMu.Lock()
	if w.running {
		w.stateMu.Unlock()
		return store.DownloadRun{}, store.ErrConflict
	}
	w.stateMu.Unlock()
	run, err := w.queue.ReserveRun(ctx, count)
	if err != nil {
		return store.DownloadRun{}, err
	}
	w.launch(run)
	return run, nil
}

// Pause prevents the next source in an active batch from starting.
func (w *Worker) Pause(ctx context.Context) error {
	return w.queue.SetPaused(ctx, true)
}

// Resume clears the persisted pause and restarts any active batch.
func (w *Worker) Resume(ctx context.Context) error {
	if err := w.queue.SetPaused(ctx, false); err != nil {
		return err
	}
	run, err := w.queue.GetActiveRun(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	w.launch(run)
	return nil
}

func (w *Worker) launch(run store.DownloadRun) {
	w.stateMu.Lock()
	if w.running {
		w.stateMu.Unlock()
		return
	}
	w.running = true
	w.runs.Add(1)
	w.stateMu.Unlock()
	go w.process(run)
}

// Wait waits for all batches started by this worker to finish.
func (w *Worker) Wait() {
	w.runs.Wait()
}

// process claims and handles every source reserved for a run serially.
func (w *Worker) process(run store.DownloadRun) {
	defer func() {
		w.stateMu.Lock()
		w.running = false
		w.stateMu.Unlock()
		w.runs.Done()
	}()
	failed, attempted := false, false
	for {
		if attempted {
			pending, err := w.queue.HasPendingRunSource(w.ctx, run.ID)
			if err != nil {
				w.closeRun(run.ID, "partial", "worker could not inspect pending sources")
				return
			}
			if !pending {
				break
			}
			capacity, err := w.queue.HasAttemptCapacity(w.ctx, w.options.MaxAttemptsPer24Hours)
			if err != nil {
				w.closeRun(run.ID, "partial", "worker could not inspect the daily attempt quota")
				return
			}
			if !capacity {
				w.closeRun(run.ID, "partial", "24-hour attempt quota reached")
				return
			}
			if err := w.wait(w.ctx, w.pause(w.options.PauseMin, w.options.PauseMax)); err != nil {
				return
			}
		}
		source, attempt, err := w.queue.StartNext(w.ctx, run.ID, w.options.MaxAttemptsPer24Hours)
		if errors.Is(err, store.ErrNoPending) {
			break
		}
		if errors.Is(err, store.ErrPaused) || w.ctx.Err() != nil {
			return
		}
		if errors.Is(err, store.ErrQuotaExceeded) {
			w.closeRun(run.ID, "partial", "24-hour attempt quota reached")
			return
		}
		if err != nil {
			w.closeRun(run.ID, "partial", "worker could not claim the next source")
			return
		}
		attempted = true
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
	template := filepath.Join(w.options.AudioDir, source.VideoID+".%(ext)s")
	metadata, err := w.downloader.Download(w.ctx, source.VideoID, template, w.options.Limits)
	if err != nil {
		return true, w.recordFailure(attempt.ID, downloadFailureCode(err), err.Error())
	}
	audio, err := audioResult(w.options.AudioDir, source.VideoID, metadata)
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

// waitForPause waits for the requested interval or until the service stops.
func waitForPause(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// randomPause returns an inclusive random whole-second duration from the range.
func randomPause(minimum, maximum time.Duration) time.Duration {
	if maximum <= minimum {
		return minimum
	}
	minimumSeconds := int64(minimum / time.Second)
	maximumSeconds := int64(maximum / time.Second)
	return time.Duration(minimumSeconds+rand.Int63n(maximumSeconds-minimumSeconds+1)) * time.Second
}
