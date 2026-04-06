package profiles

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"janus/internal/core/generator/enhanced"
	"janus/internal/database/sqlite"
)

// newTestStore creates a SQLiteStore backed by a fresh temp database.
func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("db.Connect: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewSQLiteStore(db)
}

// testProfile returns a ready-to-store Profile with unique ID and name.
func testProfile(name string) *Profile {
	now := time.Now().UTC().Truncate(time.Second)
	return &Profile{
		ID:          uuid.New().String(),
		Name:        name,
		Description: "test description",
		Options: enhanced.QuickGenerateOptions{
			FileCount:     10,
			FileSizeMin:   "1KB",
			FileSizeMax:   "1MB",
			PIIPercent:    10,
			PIIType:       "standard",
			FillerPercent: 90,
			Formats:       []string{"csv", "json"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestSQLiteStore_Create(t *testing.T) {
	s := newTestStore(t)
	p := testProfile("create-test")

	if err := s.Create(p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(p.ID)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("ID: got %q, want %q", got.ID, p.ID)
	}
	if got.Name != p.Name {
		t.Errorf("Name: got %q, want %q", got.Name, p.Name)
	}
	if got.Description != p.Description {
		t.Errorf("Description: got %q", got.Description)
	}
	if got.Options.FileCount != p.Options.FileCount {
		t.Errorf("Options.FileCount: got %d, want %d", got.Options.FileCount, p.Options.FileCount)
	}
	if got.Options.PIIType != p.Options.PIIType {
		t.Errorf("Options.PIIType: got %q, want %q", got.Options.PIIType, p.Options.PIIType)
	}
}

func TestSQLiteStore_Get_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID, got nil")
	}
}

func TestSQLiteStore_GetByName(t *testing.T) {
	s := newTestStore(t)
	p := testProfile("byname-test")
	s.Create(p)

	got, err := s.GetByName("byname-test")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got == nil {
		t.Fatal("expected profile, got nil")
	}
	if got.ID != p.ID {
		t.Errorf("ID: got %q, want %q", got.ID, p.ID)
	}
}

func TestSQLiteStore_GetByName_NotFound(t *testing.T) {
	s := newTestStore(t)
	got, err := s.GetByName("no-such-name")
	if err != nil {
		t.Fatalf("GetByName: unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing name, got %+v", got)
	}
}

func TestSQLiteStore_List(t *testing.T) {
	s := newTestStore(t)

	names := []string{"zebra", "apple", "mango"}
	for _, n := range names {
		if err := s.Create(testProfile(n)); err != nil {
			t.Fatalf("Create %q: %v", n, err)
		}
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(list))
	}
	// Should be sorted ascending by name
	if list[0].Name != "apple" || list[1].Name != "mango" || list[2].Name != "zebra" {
		t.Errorf("unexpected sort order: %q %q %q", list[0].Name, list[1].Name, list[2].Name)
	}
}

func TestSQLiteStore_List_Empty(t *testing.T) {
	s := newTestStore(t)
	list, err := s.List()
	if err != nil {
		t.Fatalf("List on empty store: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(list))
	}
}

func TestSQLiteStore_Update(t *testing.T) {
	s := newTestStore(t)
	p := testProfile("update-test")
	s.Create(p)

	p.Name = "updated-name"
	p.Description = "updated description"
	p.Options.FileCount = 999
	p.UpdatedAt = time.Now().UTC().Truncate(time.Second).Add(time.Second)

	if err := s.Update(p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Get(p.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Name != "updated-name" {
		t.Errorf("Name: got %q, want updated-name", got.Name)
	}
	if got.Description != "updated description" {
		t.Errorf("Description: got %q", got.Description)
	}
	if got.Options.FileCount != 999 {
		t.Errorf("Options.FileCount: got %d, want 999", got.Options.FileCount)
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	s := newTestStore(t)
	p := testProfile("delete-test")
	s.Create(p)

	if err := s.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := s.Get(p.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestSQLiteStore_DuplicateName(t *testing.T) {
	s := newTestStore(t)

	p1 := testProfile("dup-name")
	if err := s.Create(p1); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	p2 := testProfile("dup-name") // same name, different ID
	err := s.Create(p2)
	if err == nil {
		t.Fatal("expected UNIQUE constraint error for duplicate name, got nil")
	}
}

func TestSQLiteStore_JSONSerialization(t *testing.T) {
	s := newTestStore(t)
	p := testProfile("json-test")
	p.Options.Formats = []string{"csv", "json", "txt"}
	p.Options.PIIType = "healthcare"
	p.Options.TotalSize = "5GB"
	p.Options.FileCount = 0

	s.Create(p)

	got, _ := s.Get(p.ID)
	if len(got.Options.Formats) != 3 {
		t.Errorf("Formats: got %v, want [csv json txt]", got.Options.Formats)
	}
	if got.Options.PIIType != "healthcare" {
		t.Errorf("PIIType: got %q", got.Options.PIIType)
	}
	if got.Options.TotalSize != "5GB" {
		t.Errorf("TotalSize: got %q", got.Options.TotalSize)
	}
}
