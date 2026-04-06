package profiles

import (
	"path/filepath"
	"strings"
	"testing"

	"janus/internal/core/generator/enhanced"
	"janus/internal/database/sqlite"
)

// newTestManager creates a Manager backed by a fresh in-process SQLite database.
func newTestManager(t *testing.T) *Manager {
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
	return New(NewSQLiteStore(db))
}

// minOpts returns the minimum valid QuickGenerateOptions for a given output dir.
func minOpts(outputDir string) enhanced.QuickGenerateOptions {
	return enhanced.QuickGenerateOptions{
		OutputPath:     filepath.Join(outputDir, "payload"),
		FileCount:      10,
		FileSizeMin:    "1KB",
		FileSizeMax:    "1MB",
		PIIPercent:     10,
		PIIType:        "standard",
		FillerPercent:  90,
		Formats:        []string{"csv"},
		DirectoryDepth: 3,
	}
}

func TestManager_Create(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())

	p, err := mgr.Create("my-profile", "a description", opts)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
	if p.Name != "my-profile" {
		t.Errorf("Name: got %q, want %q", p.Name, "my-profile")
	}
	if p.Description != "a description" {
		t.Errorf("Description: got %q", p.Description)
	}
	if p.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if p.Options.Name != "my-profile" {
		t.Errorf("Options.Name should mirror profile name, got %q", p.Options.Name)
	}
}

func TestManager_Create_EmptyName(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.Create("", "desc", enhanced.QuickGenerateOptions{})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestManager_Create_DuplicateName(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())

	if _, err := mgr.Create("dup", "", opts); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := mgr.Create("dup", "", opts)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestManager_Get(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())

	created, _ := mgr.Create("get-test", "", opts)

	got, err := mgr.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID: got %q, want %q", got.ID, created.ID)
	}
	if got.Name != "get-test" {
		t.Errorf("Name: got %q", got.Name)
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.Get("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestManager_GetByName(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())
	mgr.Create("named-profile", "desc", opts)

	p, err := mgr.GetByName("named-profile")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if p.Name != "named-profile" {
		t.Errorf("Name: got %q", p.Name)
	}
}

func TestManager_GetByName_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	_, err := mgr.GetByName("does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent name")
	}
}

func TestManager_List(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if _, err := mgr.Create(name, "", opts); err != nil {
			t.Fatalf("Create %q: %v", name, err)
		}
	}

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(list))
	}
	// Store returns profiles sorted by name
	if list[0].Name != "alpha" || list[1].Name != "beta" || list[2].Name != "gamma" {
		t.Errorf("unexpected order: %v, %v, %v", list[0].Name, list[1].Name, list[2].Name)
	}
}

func TestManager_Update_Description(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())
	created, _ := mgr.Create("update-test", "original", opts)

	updated, err := mgr.Update(created.ID, map[string]interface{}{
		"description": "updated",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Description != "updated" {
		t.Errorf("Description: got %q, want %q", updated.Description, "updated")
	}
	if updated.Name != "update-test" {
		t.Errorf("Name should be unchanged, got %q", updated.Name)
	}
}

func TestManager_Update_Name(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())
	created, _ := mgr.Create("old-name", "", opts)

	updated, err := mgr.Update(created.ID, map[string]interface{}{"name": "new-name"})
	if err != nil {
		t.Fatalf("Update name: %v", err)
	}
	if updated.Name != "new-name" {
		t.Errorf("Name: got %q, want new-name", updated.Name)
	}
}

func TestManager_Update_DuplicateName(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())
	mgr.Create("existing", "", opts)
	p2, _ := mgr.Create("to-rename", "", opts)

	_, err := mgr.Update(p2.ID, map[string]interface{}{"name": "existing"})
	if err == nil {
		t.Fatal("expected error when renaming to existing name")
	}
}

func TestManager_Delete(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())
	created, _ := mgr.Create("to-delete", "", opts)

	if err := mgr.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := mgr.Get(created.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestManager_Resolve_NoOverrides(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())
	created, _ := mgr.Create("resolve-test", "", opts)

	resolved, err := mgr.Resolve(created.ID, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.FileCount != opts.FileCount {
		t.Errorf("FileCount: got %d, want %d", resolved.FileCount, opts.FileCount)
	}
}

func TestManager_Resolve_WithOverrides(t *testing.T) {
	mgr := newTestManager(t)
	opts := minOpts(t.TempDir())
	created, _ := mgr.Create("resolve-override", "", opts)

	resolved, err := mgr.Resolve(created.ID, map[string]interface{}{
		"file_count": float64(999),
	})
	if err != nil {
		t.Fatalf("Resolve with overrides: %v", err)
	}
	if resolved.FileCount != 999 {
		t.Errorf("FileCount after override: got %d, want 999", resolved.FileCount)
	}
	// Original options unchanged
	original, _ := mgr.Get(created.ID)
	if original.Options.FileCount != opts.FileCount {
		t.Error("Resolve should not mutate the stored profile")
	}
}
