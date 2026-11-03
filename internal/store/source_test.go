package store

import (
	"context"
	"errors"
	"path/filepath"
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
