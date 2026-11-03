package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestApplyMigrationsIsRepeatableAndPreservesData(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()

	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	var initialMigrations int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&initialMigrations); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sources (video_id, url, status, created_at, updated_at) VALUES (?, ?, 'pending', ?, ?)`, "abc123def45", "https://www.youtube.com/watch?v=abc123def45", "2026-11-03T00:00:00Z", "2026-11-03T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("second migration run: %v", err)
	}
	var sources, migrations, workerRows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources`).Scan(&sources); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM worker_state`).Scan(&workerRows); err != nil {
		t.Fatal(err)
	}
	if sources != 1 || migrations != initialMigrations || workerRows != 1 {
		t.Fatalf("after replay: sources=%d migrations=%d workerRows=%d, want source and worker rows preserved and migration count unchanged from %d", sources, migrations, workerRows, initialMigrations)
	}
}

func TestMigrationEnforcesSourceAndRunConstraints(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	insert := `INSERT INTO sources (video_id, url, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	if _, err := db.ExecContext(ctx, insert, "abc123def45", "https://www.youtube.com/watch?v=abc123def45", "pending", "2026-11-03T00:00:00Z", "2026-11-03T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, insert, "abc123def45", "https://www.youtube.com/watch?v=abc123def45", "pending", "2026-11-03T00:00:00Z", "2026-11-03T00:00:00Z"); err == nil {
		t.Fatal("duplicate video_id was accepted")
	}
	if _, err := db.ExecContext(ctx, insert, "differentID1", "https://www.youtube.com/watch?v=differentID1", "queued", "2026-11-03T00:00:00Z", "2026-11-03T00:00:00Z"); err == nil {
		t.Fatal("invalid source status was accepted")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO download_runs (requested_count, reserved_count, status, created_at) VALUES (0, 0, 'active', ?)`, "2026-11-03T00:00:00Z"); err == nil {
		t.Fatal("zero-sized run was accepted")
	}
}

func TestMigrationCreatesQueueAndQuotaIndexes(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ApplyMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"idx_sources_status_created", "idx_attempts_started", "idx_run_sources_run_position"} {
		var found int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&found); err != nil {
			t.Fatal(err)
		}
		if found != 1 {
			t.Errorf("missing index %s", name)
		}
	}
}

func TestMigrationDefinesForeignKeysAndSingletonWorkerState(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO download_run_sources (run_id, video_id, position) VALUES (999, 'abc123def45', 1)`); err == nil {
		t.Fatal("run source without parent run was accepted")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO worker_state (id, is_paused, updated_at) VALUES (2, 0, '2026-11-03T00:00:00Z')`); err == nil {
		t.Fatal("second worker_state row was accepted")
	}
}
