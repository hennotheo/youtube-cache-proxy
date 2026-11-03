package store

import (
	"context"
	"errors"
	"testing"

	"github.com/hennotheo/youtube-cache-proxy/internal/youtube"
)

func TestCompleteAttemptMakesSourceReadyAndRecordsSuccess(t *testing.T) {
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
	_, attempt, err := s.StartNext(ctx, run.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteAttempt(ctx, attempt.ID, AudioResult{Title: "A title", DurationSeconds: 120, Path: "audio/a.m4a", Mime: "audio/mp4", SizeBytes: 1234}); err != nil {
		t.Fatal(err)
	}
	ready, err := s.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Status != "ready" || ready.AudioPath.String != "audio/a.m4a" || ready.AudioSizeBytes.Int64 != 1234 || ready.Title.String != "A title" {
		t.Fatalf("source was not completed: status=%q path=%q size=%d title=%q", ready.Status, ready.AudioPath.String, ready.AudioSizeBytes.Int64, ready.Title.String)
	}
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM download_attempts WHERE id = ?`, attempt.ID).Scan(&status); err != nil || status != "succeeded" {
		t.Fatalf("attempt status=%q error=%v", status, err)
	}
	if err := s.CompleteAttempt(ctx, attempt.ID, AudioResult{Path: "audio/b.m4a", Mime: "audio/mp4", SizeBytes: 1}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second completion error=%v, want ErrConflict", err)
	}
}

func TestCompleteAttemptKeepsPreviouslyFetchedMetadataWhenFieldsAreOmitted(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	source, err := s.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateSourceMetadata(ctx, source.ID, youtube.VideoMetadata{
		ChannelName: "A channel", ThumbnailURL: "https://example.test/thumb.jpg", Tags: []string{"tag"},
	}); err != nil {
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
	if err := s.CompleteAttempt(ctx, attempt.ID, AudioResult{
		Title: "A title", DurationSeconds: 12, Path: "audio/a.m4a", Mime: "audio/mp4", SizeBytes: 1234,
	}); err != nil {
		t.Fatal(err)
	}
	completed, err := s.GetSource(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.ChannelName.String != "A channel" || completed.ThumbnailURL.String != "https://example.test/thumb.jpg" || completed.TagsJSON.String != `["tag"]` {
		t.Fatalf("previous metadata was cleared by an incomplete refresh: %+v", completed)
	}
}

func TestFailedAttemptCanRetryWithinRunOrBeRetriedManually(t *testing.T) {
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
	_, first, err := s.StartNext(ctx, run.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FailAttempt(ctx, first.ID, "temporary", "network timeout", true); err != nil {
		t.Fatal(err)
	}
	_, second, err := s.StartNext(ctx, run.ID, 50)
	if err != nil || second.AttemptNumber != 2 {
		t.Fatalf("second attempt number=%d error=%v", second.AttemptNumber, err)
	}
	if err := s.FailAttempt(ctx, second.ID, "private", "video private", false); err != nil {
		t.Fatal(err)
	}
	failed, err := s.GetSource(ctx, source.ID)
	if err != nil || failed.Status != "failed" || failed.ErrorCode.String != "private" || failed.AttemptCount != 2 {
		t.Fatalf("failed source status=%q code=%q attempts=%d error=%v", failed.Status, failed.ErrorCode.String, failed.AttemptCount, err)
	}
	if err := s.RetrySource(ctx, source.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("retry inside active run error=%v, want ErrConflict", err)
	}
	if err := s.CloseRun(ctx, run.ID, "completed", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RetrySource(ctx, source.ID); err != nil {
		t.Fatal(err)
	}
	retried, err := s.GetSource(ctx, source.ID)
	if err != nil || retried.Status != "pending" || retried.AttemptCount != 2 {
		t.Fatalf("retried source status=%q attempts=%d error=%v", retried.Status, retried.AttemptCount, err)
	}
}
