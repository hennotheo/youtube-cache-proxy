package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite queue and applies its pending migrations.
func Open(ctx context.Context, path string) (*Store, error) {
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
	store := &Store{db: db}
	if err := store.RecoverInterrupted(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}
