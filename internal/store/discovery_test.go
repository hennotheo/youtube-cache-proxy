package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDiscoveryPersistence(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "cache.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	q, err := s.CreateSearch(ctx, SearchInput{Query: "test"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.ClaimSearch(ctx)
	if err != nil || job.ID != q.ID {
		t.Fatalf("%+v %v", job, err)
	}
	v := DiscoveryVideo{VideoID: "abc123def45", Title: "title", ChannelID: "channel", PublishedAt: "2026-01-01T00:00:00Z", Duration: "PT3M"}
	if err = s.CompleteSearch(ctx, q.ID, []DiscoveryVideo{v, v}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		items, e := s.ListDiscoveries(ctx, DiscoveryFilter{Limit: 1, SearchID: q.ID})
		if e != nil || len(items) != 1 {
			t.Fatalf("%+v %v", items, e)
		}
	}
	items, err := s.ListDiscoveries(ctx, DiscoveryFilter{Limit: 1, Offset: 1})
	if err != nil || len(items) != 0 {
		t.Fatalf("%+v %v", items, err)
	}
	pending, err := s.CreateSearch(ctx, SearchInput{Query: "restart"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClaimSearch(ctx); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	recovered, err := s.GetSearch(ctx, pending.ID)
	if err != nil || recovered.Status != "interrupted" {
		t.Fatalf("%+v %v", recovered, err)
	}
	persisted, err := s.GetSearch(ctx, q.ID)
	if err != nil || persisted.Status != "completed" {
		t.Fatalf("%+v %v", persisted, err)
	}
}

func TestDiscoveryDatesCompareInstants(t *testing.T) {
	ctx := context.Background()
	s, e := Open(ctx, filepath.Join(t.TempDir(), "cache.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	q, e := s.CreateSearch(ctx, SearchInput{Query: "date"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ClaimSearch(ctx); e != nil {
		t.Fatal(e)
	}
	if e = s.CompleteSearch(ctx, q.ID, []DiscoveryVideo{{VideoID: "abc123def45", PublishedAt: "2026-01-01T00:30:00Z"}}); e != nil {
		t.Fatal(e)
	}
	items, e := s.ListDiscoveries(ctx, DiscoveryFilter{Limit: 10, PublishedAfter: "2026-01-01T01:00:00+02:00"})
	if e != nil || len(items) != 1 {
		t.Fatalf("timezone-equivalent filtering: %+v %v", items, e)
	}
}
