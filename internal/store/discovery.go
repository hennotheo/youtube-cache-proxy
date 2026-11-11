package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type SearchInput struct {
	Query           string `json:"query"`
	ChannelID       string `json:"channelId,omitempty"`
	PublishedAfter  string `json:"publishedAfter,omitempty"`
	PublishedBefore string `json:"publishedBefore,omitempty"`
}
type Search struct {
	ID         int64       `json:"id"`
	Input      SearchInput `json:"input"`
	Status     string      `json:"status"`
	Error      string      `json:"error,omitempty"`
	CreatedAt  string      `json:"createdAt"`
	FinishedAt string      `json:"finishedAt,omitempty"`
}
type DiscoveryVideo struct {
	VideoID      string `json:"videoId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ChannelID    string `json:"channelId"`
	ChannelTitle string `json:"channelTitle"`
	PublishedAt  string `json:"publishedAt"`
	Thumbnail    string `json:"thumbnail"`
	Duration     string `json:"duration"`
	CachedAt     string `json:"cachedAt"`
}
type DiscoveryFilter struct {
	SearchID                                         int64
	ChannelID, PublishedAfter, PublishedBefore, Sort string
	Limit, Offset                                    int
}

func (s *Store) CreateSearch(ctx context.Context, input SearchInput) (Search, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return Search{}, err
	}
	created := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, "INSERT INTO discovery_searches(input,created_at) VALUES(?,?)", string(raw), created)
	if err != nil {
		return Search{}, err
	}
	id, err := result.LastInsertId()
	return Search{ID: id, Input: input, Status: "queued", CreatedAt: created}, err
}
func scanSearch(row interface{ Scan(...any) error }) (Search, error) {
	var q Search
	var raw string
	err := row.Scan(&q.ID, &raw, &q.Status, &q.Error, &q.CreatedAt, &q.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return q, ErrNotFound
	}
	if err != nil {
		return q, err
	}
	err = json.Unmarshal([]byte(raw), &q.Input)
	return q, err
}
func (s *Store) GetSearch(ctx context.Context, id int64) (Search, error) {
	return scanSearch(s.db.QueryRowContext(ctx, "SELECT id,input,status,error,created_at,finished_at FROM discovery_searches WHERE id=?", id))
}
func (s *Store) ClaimSearch(ctx context.Context) (Search, error) {
	return scanSearch(s.db.QueryRowContext(ctx, `UPDATE discovery_searches SET status='running' WHERE id=(SELECT id FROM discovery_searches WHERE status='queued' ORDER BY id LIMIT 1) AND status='queued' RETURNING id,input,status,error,created_at,finished_at`))
}
func (s *Store) RecoverSearches(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "UPDATE discovery_searches SET status='interrupted',error='service restarted',finished_at=? WHERE status='running'", time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) FailSearch(ctx context.Context, id int64, message string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE discovery_searches SET status='failed',error=?,finished_at=? WHERE id=? AND status='running'", message, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}
func (s *Store) CompleteSearch(ctx context.Context, id int64, videos []DiscoveryVideo) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, v := range videos {
		v.CachedAt = now
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO discovery_videos(video_id,channel_id,published_at,metadata) VALUES(?,?,?,?) ON CONFLICT(video_id) DO UPDATE SET channel_id=excluded.channel_id,published_at=excluded.published_at,metadata=excluded.metadata`, v.VideoID, v.ChannelID, canonicalDiscoveryTime(v.PublishedAt), string(raw)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO discovery_results(search_id,video_id) VALUES(?,?)", id, v.VideoID); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, "UPDATE discovery_searches SET status='completed',finished_at=? WHERE id=? AND status='running'", now, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return tx.Commit()
}
func (s *Store) ListDiscoveries(ctx context.Context, f DiscoveryFilter) ([]DiscoveryVideo, error) {
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 {
		return nil, ErrConflict
	}
	order := "v.published_at DESC,v.video_id"
	if f.Sort == "publishedAt" {
		order = "v.published_at ASC,v.video_id"
	} else if f.Sort != "" && f.Sort != "-publishedAt" {
		return nil, ErrConflict
	}
	query := "SELECT v.metadata FROM discovery_videos v WHERE 1=1"
	args := []any{}
	if f.SearchID > 0 {
		query += " AND EXISTS(SELECT 1 FROM discovery_results r WHERE r.video_id=v.video_id AND r.search_id=?)"
		args = append(args, f.SearchID)
	}
	if f.ChannelID != "" {
		query += " AND v.channel_id=?"
		args = append(args, f.ChannelID)
	}
	if f.PublishedAfter != "" {
		query += " AND v.published_at>?"
		args = append(args, canonicalDiscoveryTime(f.PublishedAfter))
	}
	if f.PublishedBefore != "" {
		query += " AND v.published_at<?"
		args = append(args, canonicalDiscoveryTime(f.PublishedBefore))
	}
	query += " ORDER BY " + order + " LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DiscoveryVideo, 0)
	for rows.Next() {
		var raw string
		var v DiscoveryVideo
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

// Fixed-width UTC timestamps preserve chronological order in indexed TEXT columns.
func canonicalDiscoveryTime(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.UTC().Format("2006-01-02T15:04:05.000000000Z")
}
