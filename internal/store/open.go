package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/gofrs/flock"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite queue and applies its pending migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SQLite path: %w", err)
	}
	// The lock is held until Close, so a second instance cannot recover a live run.
	fileLock := flock.New(absolute + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("lock SQLite queue: %w", err)
	}
	if !locked {
		return nil, ErrConflict
	}
	release := true
	defer func() {
		if release {
			_ = fileLock.Unlock()
		}
	}()
	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, fmt.Errorf("prepare SQLite path: %w", err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	// One connection serializes the low-volume queue's read-modify-write operations.
	db.SetMaxOpenConns(1)
	if err := ApplyMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &Store{db: db, lock: fileLock}
	if err := store.RecoverInterrupted(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.RecoverSearches(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	release = false
	return store, nil
}
