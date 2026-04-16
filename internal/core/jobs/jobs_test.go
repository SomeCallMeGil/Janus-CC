package jobs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// ---- Run ID tests ----

func TestGenerateRunID_Format(t *testing.T) {
	id := GenerateRunID()
	parts := strings.Split(id, "-")
	if len(parts) != 3 {
		t.Fatalf("expected 3 dash-separated parts, got %d: %q", len(parts), id)
	}
	if len(parts[0]) != 8 {
		t.Errorf("date part should be 8 chars (YYYYMMDD), got %q", parts[0])
	}
	if len(parts[1]) != 4 {
		t.Errorf("time part should be 4 chars (HHMM), got %q", parts[1])
	}
	if len(parts[2]) != 8 {
		t.Errorf("random part should be 8 hex chars, got %q", parts[2])
	}
}

func TestGenerateRunID_Unique(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		id := GenerateRunID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate run ID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

// ---- Payload directory tests ----

func TestCreatePayloadDirectory(t *testing.T) {
	baseDir := t.TempDir()
	runID := "20260415-1430-aabbccdd"

	payloadPath, err := CreatePayloadDirectory(runID, baseDir)
	if err != nil {
		t.Fatalf("CreatePayloadDirectory: %v", err)
	}

	expectedRoot := filepath.Join(baseDir, "payloads", runID)
	if payloadPath != expectedRoot {
		t.Errorf("payloadPath: got %q, want %q", payloadPath, expectedRoot)
	}

	for _, sub := range []string{"", "output", "logs"} {
		dir := filepath.Join(expectedRoot, sub)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Errorf("expected directory %q to exist", dir)
		}
	}
}

func TestCreatePayloadDirectory_Idempotent(t *testing.T) {
	baseDir := t.TempDir()
	runID := "20260415-1430-aabbccdd"

	for i := 0; i < 3; i++ {
		if _, err := CreatePayloadDirectory(runID, baseDir); err != nil {
			t.Fatalf("call %d: CreatePayloadDirectory: %v", i+1, err)
		}
	}
}

func TestSavePayloadMetadata(t *testing.T) {
	baseDir := t.TempDir()
	runID := "20260415-1430-aabbccdd"
	payloadPath, _ := CreatePayloadDirectory(runID, baseDir)

	cfg, _ := json.Marshal(map[string]interface{}{"file_count": 100})
	profileJSON, _ := json.Marshal(map[string]string{"name": "test-profile"})

	job := &JobRun{
		ID:         runID,
		ProfileID:  "profile-abc",
		Config:     json.RawMessage(cfg),
		StartTime:  time.Now(),
		Status:     JobRunning,
		OutputPath: filepath.Join(payloadPath, "output"),
		CreatedAt:  time.Now(),
	}

	if err := SavePayloadMetadata(payloadPath, job, profileJSON); err != nil {
		t.Fatalf("SavePayloadMetadata: %v", err)
	}

	for _, fname := range []string{"config.json", "profile.json", "metadata.json"} {
		path := filepath.Join(payloadPath, fname)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %q to exist: %v", fname, err)
		}
	}

	// Validate metadata.json content
	raw, _ := os.ReadFile(filepath.Join(payloadPath, "metadata.json"))
	var meta PayloadMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("unmarshal metadata.json: %v", err)
	}
	if meta.RunID != runID {
		t.Errorf("RunID: got %q, want %q", meta.RunID, runID)
	}
	if meta.ProfileID != "profile-abc" {
		t.Errorf("ProfileID: got %q", meta.ProfileID)
	}
}

func TestSavePayloadMetadata_NoProfile(t *testing.T) {
	baseDir := t.TempDir()
	payloadPath, _ := CreatePayloadDirectory("run-x", baseDir)

	job := &JobRun{
		ID:        "run-x",
		StartTime: time.Now(),
		Status:    JobRunning,
		CreatedAt: time.Now(),
	}

	if err := SavePayloadMetadata(payloadPath, job, nil); err != nil {
		t.Fatalf("SavePayloadMetadata without profile: %v", err)
	}

	// profile.json should NOT be created
	if _, err := os.Stat(filepath.Join(payloadPath, "profile.json")); !os.IsNotExist(err) {
		t.Error("profile.json should not exist when no profile JSON provided")
	}
	// metadata.json MUST exist
	if _, err := os.Stat(filepath.Join(payloadPath, "metadata.json")); err != nil {
		t.Error("metadata.json should exist")
	}
}

// ---- SQLiteStore CRUD tests ----

func testJobRun(id string) *JobRun {
	cfg, _ := json.Marshal(map[string]string{"output": "/tmp/" + id})
	return &JobRun{
		ID:         id,
		ProfileID:  "prof-1",
		Config:     json.RawMessage(cfg),
		StartTime:  time.Now().UTC().Truncate(time.Second),
		Status:     JobRunning,
		OutputPath: "/tmp/" + id,
		CreatedAt:  time.Now().UTC().Truncate(time.Second),
	}
}

func TestSQLiteStore_Create(t *testing.T) {
	s := newTestStore(t)
	job := testJobRun("run-create")

	if err := s.Create(job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(job.ID)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.ID != job.ID {
		t.Errorf("ID: got %q, want %q", got.ID, job.ID)
	}
	if got.ProfileID != job.ProfileID {
		t.Errorf("ProfileID: got %q", got.ProfileID)
	}
	if got.Status != JobRunning {
		t.Errorf("Status: got %q, want %q", got.Status, JobRunning)
	}
	if got.OutputPath != job.OutputPath {
		t.Errorf("OutputPath: got %q", got.OutputPath)
	}
}

func TestSQLiteStore_Get_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get("nonexistent-run-id")
	if err == nil {
		t.Fatal("expected error for nonexistent ID, got nil")
	}
}

func TestSQLiteStore_Update(t *testing.T) {
	s := newTestStore(t)
	job := testJobRun("run-update")
	s.Create(job)

	end := time.Now().UTC().Truncate(time.Second)
	job.EndTime = &end
	job.Status = JobCompleted
	job.FileCount = 42
	job.ErrorLog = ""

	if err := s.Update(job); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Get(job.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Status != JobCompleted {
		t.Errorf("Status: got %q, want %q", got.Status, JobCompleted)
	}
	if got.FileCount != 42 {
		t.Errorf("FileCount: got %d, want 42", got.FileCount)
	}
	if got.EndTime == nil {
		t.Error("EndTime: got nil, want non-nil")
	}
}

func TestSQLiteStore_Delete(t *testing.T) {
	s := newTestStore(t)
	job := testJobRun("run-delete")
	s.Create(job)

	if err := s.Delete(job.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(job.ID); err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestSQLiteStore_List(t *testing.T) {
	s := newTestStore(t)

	j1 := testJobRun("run-list-1")
	j1.ProfileID = "profile-A"
	j2 := testJobRun("run-list-2")
	j2.ProfileID = "profile-A"
	j3 := testJobRun("run-list-3")
	j3.ProfileID = "profile-B"
	j3.Status = JobCompleted

	for _, j := range []*JobRun{j1, j2, j3} {
		s.Create(j)
	}

	// List all
	all, err := s.List(JobFilters{})
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 runs, got %d", len(all))
	}

	// Filter by profile
	byProfile, err := s.List(JobFilters{ProfileID: "profile-A"})
	if err != nil {
		t.Fatalf("List by profile: %v", err)
	}
	if len(byProfile) != 2 {
		t.Errorf("expected 2 runs for profile-A, got %d", len(byProfile))
	}

	// Filter by status
	byStatus, err := s.List(JobFilters{Status: JobCompleted})
	if err != nil {
		t.Fatalf("List by status: %v", err)
	}
	if len(byStatus) != 1 {
		t.Errorf("expected 1 completed run, got %d", len(byStatus))
	}

	// Limit
	limited, err := s.List(JobFilters{Limit: 1})
	if err != nil {
		t.Fatalf("List with limit: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("expected 1 run with limit=1, got %d", len(limited))
	}
}

// ---- Lifecycle helpers tests ----

func TestStartRun_FinishRun(t *testing.T) {
	s := newTestStore(t)

	cfg, _ := json.Marshal(map[string]string{"path": "/tmp/test"})
	job, err := StartRun(s, "prof-1", cfg, "/tmp/test")
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if job.Status != JobRunning {
		t.Errorf("Status after StartRun: got %q, want %q", job.Status, JobRunning)
	}
	if job.ID == "" {
		t.Error("ID should not be empty after StartRun")
	}

	if err := FinishRun(s, job, JobCompleted, 50, ""); err != nil {
		t.Fatalf("FinishRun: %v", err)
	}

	got, err := s.Get(job.ID)
	if err != nil {
		t.Fatalf("Get after FinishRun: %v", err)
	}
	if got.Status != JobCompleted {
		t.Errorf("Status after FinishRun: got %q, want %q", got.Status, JobCompleted)
	}
	if got.FileCount != 50 {
		t.Errorf("FileCount: got %d, want 50", got.FileCount)
	}
	if got.EndTime == nil {
		t.Error("EndTime should be set after FinishRun")
	}
}

func TestStartRun_Failed(t *testing.T) {
	s := newTestStore(t)

	job, _ := StartRun(s, "", nil, "/tmp/x")
	if err := FinishRun(s, job, JobFailed, 0, "disk full"); err != nil {
		t.Fatalf("FinishRun with failure: %v", err)
	}

	got, _ := s.Get(job.ID)
	if got.Status != JobFailed {
		t.Errorf("Status: got %q, want %q", got.Status, JobFailed)
	}
	if got.ErrorLog != "disk full" {
		t.Errorf("ErrorLog: got %q, want %q", got.ErrorLog, "disk full")
	}
}
