package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestDeleteSourceConflictsWhileItsRunIsActiveAndSucceedsAfterClosure(t *testing.T) {
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
	if err := s.DeleteSource(ctx, source.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete while run is active = %v, want ErrConflict", err)
	}
	if _, err := s.GetSource(ctx, source.ID); err != nil {
		t.Fatalf("source was deleted during active run: %v", err)
	}
	if err := s.CloseRun(ctx, run.ID, "completed", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSource(ctx, source.ID); err != nil {
		t.Fatalf("delete after run closure = %v, want success", err)
	}
	if _, err := s.GetSource(ctx, source.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("lookup after deletion = %v, want ErrNotFound", err)
	}

	var snapshotID sql.NullInt64
	var snapshotVideoID string
	if err := s.db.QueryRowContext(ctx, `SELECT source_id, video_id FROM download_run_sources WHERE run_id = ?`, run.ID).Scan(&snapshotID, &snapshotVideoID); err != nil {
		t.Fatal(err)
	}
	if snapshotID.Valid || snapshotVideoID != source.VideoID {
		t.Fatalf("run snapshot after deletion: source ID valid=%t, video ID=%q", snapshotID.Valid, snapshotVideoID)
	}
}

func TestDeleteSourceReturnsNotFoundForUnknownID(t *testing.T) {
	s := openTestStore(t)
	if err := s.DeleteSource(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown source = %v, want ErrNotFound", err)
	}
}
