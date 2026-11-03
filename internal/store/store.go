package store

import (
	"database/sql"
	"errors"

	"github.com/gofrs/flock"
)

// Store owns the SQLite connection used by queue operations.
type Store struct {
	db   *sql.DB
	lock *flock.Flock
}

// Close releases the database connection.
func (s *Store) Close() error { return errors.Join(s.db.Close(), s.lock.Unlock()) }
