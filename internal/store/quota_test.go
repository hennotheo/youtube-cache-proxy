package store

import (
	"context"
	"errors"
	"testing"
)

func TestStartNextCountsRetriesTowardRollingQuota(t *testing.T) {
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
	_, attempt, err := s.StartNext(ctx, run.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FailAttempt(ctx, attempt.ID, "temporary", "network timeout", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.StartNext(ctx, run.ID, 1); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("retry start error = %v, want ErrQuotaExceeded", err)
	}
	unchanged, err := s.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Status != "pending" || unchanged.AttemptCount != 1 {
		t.Fatalf("source after rejected retry: status=%q attempts=%d, want pending and one attempt", unchanged.Status, unchanged.AttemptCount)
	}
	var attempts int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM download_attempts WHERE video_id = ?`, source.VideoID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempt rows = %d, want 1 after quota rejection", attempts)
	}
}

func TestStartNextExcludesAttemptsOlderThanTwentyFourHours(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	_, err := s.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	run, err := s.ReserveRun(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, first, err := s.StartNext(ctx, run.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FailAttempt(ctx, first.ID, "temporary", "network timeout", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE download_attempts SET started_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-25 hours') WHERE id = ?`, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.StartNext(ctx, run.ID, 1); err != nil {
		t.Fatalf("start after old attempt = %v, want success", err)
	}
}
