package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestSecondStoreCannotInterruptActiveOwner(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	owner, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	source, err := owner.CreateSource(ctx, "aaaaaaaaaaa", "https://www.youtube.com/watch?v=aaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	run, err := owner.ReserveRun(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := owner.StartNext(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	second, err := Open(ctx, path)
	if second != nil {
		defer second.Close()
	}
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("second Open error=%v, want ErrConflict", err)
	}
	current, err := owner.GetSource(ctx, source.ID)
	if err != nil || current.Status != "downloading" {
		t.Fatalf("active source changed: status=%q error=%v", current.Status, err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	afterRelease, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open after owner close: %v", err)
	}
	defer afterRelease.Close()
	recovered, err := afterRelease.GetSource(ctx, source.ID)
	if err != nil || recovered.Status != "pending" {
		t.Fatalf("source after restart: status=%q error=%v", recovered.Status, err)
	}
}
