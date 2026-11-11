package discovery

import (
	"context"
	"github.com/hennotheo/youtube-cache-proxy/internal/store"
	"path/filepath"
	"testing"
	"time"
)

type blockingProvider struct{ started chan struct{} }

func (p blockingProvider) Search(ctx context.Context, in store.SearchInput) ([]store.DiscoveryVideo, error) {
	close(p.started)
	<-ctx.Done()
	return nil, ctx.Err()
}
func TestWorkerShutdownAndRecovery(t *testing.T) {
	ctx := context.Background()
	s, e := store.Open(ctx, filepath.Join(t.TempDir(), "cache.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	q, e := s.CreateSearch(ctx, store.SearchInput{Query: "test"})
	if e != nil {
		t.Fatal(e)
	}
	p := blockingProvider{make(chan struct{})}
	w, e := NewWorker(ctx, s, p)
	if e != nil {
		t.Fatal(e)
	}
	select {
	case <-p.started:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not start")
	}
	w.Close()
	if e = s.RecoverSearches(ctx); e != nil {
		t.Fatal(e)
	}
	q, e = s.GetSearch(ctx, q.ID)
	if e != nil || q.Status != "interrupted" {
		t.Fatal(q, e)
	}
}
