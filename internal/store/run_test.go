package store

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestRunReservesOldestPendingSourcesAndRejectsSecondActiveRun(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	for _, id := range []string{"aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"} {
		if _, err := s.CreateSource(ctx, id, "https://www.youtube.com/watch?v="+id); err != nil {
			t.Fatal(err)
		}
	}
	run, err := s.ReserveRun(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if run.RequestedCount != 2 || run.ReservedCount != 2 || run.Status != "active" {
		t.Fatalf("unexpected run: requested=%d reserved=%d status=%q", run.RequestedCount, run.ReservedCount, run.Status)
	}
	if _, err := s.ReserveRun(ctx, 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("second run error = %v, want ErrConflict", err)
	}
	first, attempt, err := s.StartNext(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.VideoID != "aaaaaaaaaaa" || first.Status != "downloading" || first.AttemptCount != 1 || attempt.SourceID != first.ID {
		t.Fatalf("unexpected first claim: video=%q status=%q count=%d attemptSource=%d", first.VideoID, first.Status, first.AttemptCount, attempt.SourceID)
	}
	if _, _, err := s.StartNext(ctx, run.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("parallel start error = %v, want ErrConflict", err)
	}
}

func TestConcurrentRunReservationsCreateOnlyOneActiveRun(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	for _, id := range []string{"aaaaaaaaaaa", "bbbbbbbbbbb"} {
		if _, err := s.CreateSource(ctx, id, "https://www.youtube.com/watch?v="+id); err != nil {
			t.Fatal(err)
		}
	}
	results := make(chan error, 8)
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, err := s.ReserveRun(ctx, 1)
			results <- err
		}()
	}
	workers.Wait()
	close(results)
	var successful int
	for err := range results {
		if err == nil {
			successful++
		} else if !errors.Is(err, ErrConflict) {
			t.Fatalf("unexpected reservation error: %v", err)
		}
	}
	if successful != 1 {
		t.Fatalf("successful reservations=%d, want 1", successful)
	}
}

func TestReserveRunDoesNotMisreportDatabaseWriteFailureAsConflict(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	if _, err := s.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER fail_run_insert BEFORE INSERT ON download_runs
		BEGIN SELECT RAISE(ABORT, 'simulated write failure'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := s.ReserveRun(ctx, 1)
	if err == nil || errors.Is(err, ErrConflict) {
		t.Fatalf("database write error = %v, want distinct failure", err)
	}
}

func TestPauseBlocksNextAttemptWithoutChangingCurrentSource(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	source, err := s.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.ReserveRun(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPaused(ctx, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.StartNext(ctx, run.ID); !errors.Is(err, ErrPaused) {
		t.Fatalf("paused start error = %v, want ErrPaused", err)
	}
	unchanged, err := s.GetSource(ctx, source.ID)
	if err != nil || unchanged.Status != "pending" {
		t.Fatalf("paused source status = %q, error = %v", unchanged.Status, err)
	}
	if err := s.SetPaused(ctx, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.StartNext(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
}
