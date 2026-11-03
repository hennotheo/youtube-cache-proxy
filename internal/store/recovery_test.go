package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenRecoversInterruptedRunWithoutStartingWork(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	source, err := s.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.ReserveRun(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, attempt, err := s.StartNext(ctx, run.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPaused(ctx, true); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	recovered, err := reopened.GetSource(ctx, source.ID)
	if err != nil || recovered.Status != "pending" || recovered.AttemptCount != 1 {
		t.Fatalf("source after restart: status=%q count=%d error=%v", recovered.Status, recovered.AttemptCount, err)
	}
	var runStatus, attemptStatus string
	var paused int
	if err := reopened.db.QueryRowContext(ctx, `SELECT status FROM download_runs WHERE id = ?`, run.ID).Scan(&runStatus); err != nil {
		t.Fatal(err)
	}
	if err := reopened.db.QueryRowContext(ctx, `SELECT status FROM download_attempts WHERE id = ?`, attempt.ID).Scan(&attemptStatus); err != nil {
		t.Fatal(err)
	}
	if err := reopened.db.QueryRowContext(ctx, `SELECT is_paused FROM worker_state WHERE id = 1`).Scan(&paused); err != nil {
		t.Fatal(err)
	}
	if runStatus != "interrupted" || attemptStatus != "failed" || paused != 1 {
		t.Fatalf("after restart: run=%q attempt=%q paused=%d", runStatus, attemptStatus, paused)
	}
}
