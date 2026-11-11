package discovery

import (
	"context"
	"errors"
	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"time"
)

type Provider interface {
	Search(context.Context, store.SearchInput) ([]store.DiscoveryVideo, error)
}
type Worker struct {
	store    *store.Store
	provider Provider
	wake     chan struct{}
	done     chan struct{}
	cancel   context.CancelFunc
}

func NewWorker(ctx context.Context, s *store.Store, p Provider) (*Worker, error) {
	if err := s.RecoverSearches(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	w := &Worker{store: s, provider: p, wake: make(chan struct{}, 1), done: make(chan struct{}), cancel: cancel}
	go w.run(ctx)
	w.Notify()
	return w, nil
}
func (w *Worker) Notify() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}
func (w *Worker) Close() { w.cancel(); <-w.done }
func (w *Worker) run(ctx context.Context) {
	defer close(w.done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.wake:
		}
		for ctx.Err() == nil {
			job, err := w.store.ClaimSearch(ctx)
			if errors.Is(err, store.ErrNotFound) {
				break
			}
			if err != nil {
				return
			} // Pending work remains durable for restart.
			callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			videos, err := w.provider.Search(callCtx, job.Input)
			cancel()
			if ctx.Err() != nil {
				return
			} // Marked interrupted on next startup.
			if err != nil {
				err = w.store.FailSearch(ctx, job.ID, err.Error())
			} else {
				err = w.store.CompleteSearch(ctx, job.ID, videos)
			}
			if err != nil {
				return
			}
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}
