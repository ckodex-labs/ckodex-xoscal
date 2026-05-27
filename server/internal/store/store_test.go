package store

import (
	"context"
	"testing"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
)

func newTestStore(t *testing.T) Store {
	s, err := NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCatalogCRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	c := &catalogv1.Catalog{
		Uuid: &commonv1.UUID{Value: "cat-001"},
		Metadata: &commonv1.Metadata{
			Title:   "Test Catalog",
			Version: "1.0.0",
		},
	}

	if err := s.CreateCatalog(ctx, c); err != nil {
		t.Fatalf("create catalog: %v", err)
	}

	got, err := s.GetCatalog(ctx, "cat-001")
	if err != nil {
		t.Fatalf("get catalog: %v", err)
	}
	if got.Metadata.Title != "Test Catalog" {
		t.Errorf("title = %q, want %q", got.Metadata.Title, "Test Catalog")
	}

	c.Metadata.Title = "Updated Catalog"
	if err := s.UpdateCatalog(ctx, "cat-001", c); err != nil {
		t.Fatalf("update catalog: %v", err)
	}
	got, _ = s.GetCatalog(ctx, "cat-001")
	if got.Metadata.Title != "Updated Catalog" {
		t.Errorf("title after update = %q, want %q", got.Metadata.Title, "Updated Catalog")
	}

	list, _, err := s.ListCatalogs(ctx, "", 10, "")
	if err != nil {
		t.Fatalf("list catalogs: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("len(list) = %d, want 1", len(list))
	}

	if err := s.DeleteCatalog(ctx, "cat-001"); err != nil {
		t.Fatalf("delete catalog: %v", err)
	}
	_, err = s.GetCatalog(ctx, "cat-001")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestProfileCRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	p := &profilev1.Profile{
		Uuid: &commonv1.UUID{Value: "prof-001"},
		Metadata: &commonv1.Metadata{
			Title:   "Test Profile",
			Version: "1.0.0",
		},
	}

	if err := s.CreateProfile(ctx, p); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	got, err := s.GetProfile(ctx, "prof-001")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if got.Metadata.Title != "Test Profile" {
		t.Errorf("title = %q, want %q", got.Metadata.Title, "Test Profile")
	}

	list, _, err := s.ListProfiles(ctx, "", 10, "")
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("len(list) = %d, want 1", len(list))
	}

	if err := s.DeleteProfile(ctx, "prof-001"); err != nil {
		t.Fatalf("delete profile: %v", err)
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	_ = s.CreateCatalog(ctx, &catalogv1.Catalog{
		Uuid:     &commonv1.UUID{Value: "c1"},
		Metadata: &commonv1.Metadata{Title: "ISO 27001 Catalog", Version: "2023"},
	})
	_ = s.CreateProfile(ctx, &profilev1.Profile{
		Uuid:     &commonv1.UUID{Value: "p1"},
		Metadata: &commonv1.Metadata{Title: "ISO 27001 Profile", Version: "2023"},
	})

	results, _, err := s.Search(ctx, "ISO", nil, 10, "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("search results = %d, want 2", len(results))
	}

	results, _, err = s.Search(ctx, "Catalog", []string{"catalog"}, 10, "")
	if err != nil {
		t.Fatalf("search filtered: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("filtered search results = %d, want 1", len(results))
	}
}
