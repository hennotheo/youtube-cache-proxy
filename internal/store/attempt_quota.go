package store

import (
	"context"
	"database/sql"
	"fmt"
)

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// HasAttemptCapacity reports whether another attempt fits within the rolling 24-hour limit.
func (s *Store) HasAttemptCapacity(ctx context.Context, maxAttemptsPer24Hours int) (bool, error) {
	if maxAttemptsPer24Hours < 1 {
		return false, fmt.Errorf("daily attempt quota must be positive")
	}
	attempts, err := recentAttemptCount(ctx, s.db)
	if err != nil {
		return false, err
	}
	return attempts < maxAttemptsPer24Hours, nil
}

func recentAttemptCount(ctx context.Context, queryer rowQueryer) (int, error) {
	var attempts int
	err := queryer.QueryRowContext(ctx, `SELECT COUNT(*) FROM download_attempts
		WHERE started_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-24 hours')`).Scan(&attempts)
	return attempts, err
}
