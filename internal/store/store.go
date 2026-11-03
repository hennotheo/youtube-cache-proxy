package store

import "database/sql"

// Store owns the SQLite connection used by queue operations.
type Store struct {
	db *sql.DB
}

// Close releases the database connection.
func (s *Store) Close() error { return s.db.Close() }
