package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenKeepsForeignKeysOnReplacementConnection(t *testing.T) {
	s := openTestStore(t)
	s.db.SetConnMaxLifetime(time.Nanosecond)
	time.Sleep(time.Millisecond)
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO download_run_sources
		(run_id, video_id, position) VALUES (999, 'aaaaaaaaaaa', 1)`)
	if err == nil {
		t.Fatal("foreign key was not enforced after connection replacement")
	}
}

func TestCreateSourceRejectsDuplicateVideoID(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	first, err := s.CreateSource(ctx, "abc123def45", "https://www.youtube.com/watch?v=abc123def45")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID < 1 || first.Status != "pending" || first.AttemptCount != 0 {
		t.Fatalf("unexpected source: id=%d status=%q attempts=%d", first.ID, first.Status, first.AttemptCount)
	}
	if _, err := s.CreateSource(ctx, "abc123def45", "https://youtu.be/abc123def45"); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate error = %v, want ErrDuplicate", err)
	}
}

func TestListSourcesFiltersAndPaginatesNewestFirst(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	for _, id := range []string{"aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"} {
		if _, err := s.CreateSource(ctx, id, "https://www.youtube.com/watch?v="+id); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.ListSources(ctx, "pending", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].VideoID != "bbbbbbbbbbb" {
		t.Fatalf("page = %+v, want middle source", page)
	}
	if _, err := s.ListSources(ctx, "invalid", 10, 0); err == nil {
		t.Fatal("invalid status filter was accepted")
	}
}

func TestConcurrentQueueWritesUseWalAndBusyTimeout(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	var journalMode string
	if err := s.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal mode = %q, want wal", journalMode)
	}
	var busyTimeout int
	if err := s.db.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy timeout = %d, want 5000 ms", busyTimeout)
	}

	const workers = 8
	const writesPerWorker = 20
	errCh := make(chan error, workers*writesPerWorker)
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for write := range writesPerWorker {
				id := fmt.Sprintf("%011d", worker*writesPerWorker+write+1)
				_, err := s.CreateSource(ctx, id, "https://www.youtube.com/watch?v="+id)
				if err != nil {
					errCh <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent source creation failed: %v", err)
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources WHERE status = 'pending'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if got, want := count, workers*writesPerWorker; got != want {
		t.Fatalf("stored sources = %d, want %d", got, want)
	}
}
