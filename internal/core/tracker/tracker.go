// Package tracker provides file tracking and hashing functionality.
package tracker

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"janus/internal/database/models"
)

// Tracker handles file tracking and hashing
type Tracker struct {
	db models.Database
}

// New creates a new tracker
func New(db models.Database) *Tracker {
	return &Tracker{db: db}
}

// TrackFile tracks a file by calculating its hash and storing metadata
func (t *Tracker) TrackFile(scenarioID, path string, dataType string) error {
	// Calculate SHA-256 hash
	hash, err := t.HashFile(path)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	// Create file record
	file := &models.File{
		ScenarioID:       scenarioID,
		Path:             path,
		SHA256:           hash,
		Size:             info.Size(),
		Extension:        filepath.Ext(path),
		DataType:         dataType,
		EncryptionStatus: models.FileStatusPending,
		CreatedAt:        time.Now(),
	}

	return t.db.CreateFile(file)
}

// HashFile calculates the SHA-256 hash of a file
func (t *Tracker) HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// TrackDirectory recursively tracks all files in a directory
func (t *Tracker) TrackDirectory(scenarioID, root string, dataType string) (int, error) {
	count := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Track file
		if err := t.TrackFile(scenarioID, path, dataType); err != nil {
			return fmt.Errorf("track %s: %w", path, err)
		}

		count++
		return nil
	})

	return count, err
}

// VerifyIntegrity verifies the integrity of tracked files
func (t *Tracker) VerifyIntegrity(scenarioID string) ([]string, error) {
	// Get all files for scenario
	files, err := t.db.ListFilesByScenario(scenarioID, models.FileFilters{})
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	var corrupted []string

	for _, file := range files {
		// Skip encrypted files (hash will be different)
		if file.EncryptionStatus == models.FileStatusEncrypted {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(file.Path); os.IsNotExist(err) {
			corrupted = append(corrupted, fmt.Sprintf("%s: file missing", file.Path))
			continue
		}

		// Calculate current hash
		currentHash, err := t.HashFile(file.Path)
		if err != nil {
			corrupted = append(corrupted, fmt.Sprintf("%s: cannot hash: %v", file.Path, err))
			continue
		}

		// Compare hashes
		if currentHash != file.SHA256 {
			corrupted = append(corrupted, fmt.Sprintf("%s: hash mismatch", file.Path))
		}
	}

	return corrupted, nil
}

// ExportCSV exports file manifest to CSV
func (t *Tracker) ExportCSV(scenarioID, outputPath string) error {
	// Get all files for scenario
	files, err := t.db.ListFilesByScenario(scenarioID, models.FileFilters{})
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write header
	header := []string{
		"ID",
		"Path",
		"SHA256",
		"Size",
		"Extension",
		"DataType",
		"EncryptionStatus",
		"CreatedAt",
		"EncryptedAt",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write data rows
	for _, file := range files {
		encryptedAt := ""
		if file.EncryptedAt != nil {
			encryptedAt = file.EncryptedAt.Format(time.RFC3339)
		}

		row := []string{
			fmt.Sprintf("%d", file.ID),
			file.Path,
			file.SHA256,
			fmt.Sprintf("%d", file.Size),
			file.Extension,
			file.DataType,
			string(file.EncryptionStatus),
			file.CreatedAt.Format(time.RFC3339),
			encryptedAt,
		}

		if err := w.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

// UpdateFileStatus updates the encryption status of a file
func (t *Tracker) UpdateFileStatus(fileID int64, status models.FileStatus) error {
	file, err := t.db.GetFile(fileID)
	if err != nil {
		return err
	}

	file.EncryptionStatus = status
	if status == models.FileStatusEncrypted {
		now := time.Now()
		file.EncryptedAt = &now
	}

	return t.db.UpdateFile(file)
}

// GetFilesByStatus returns files with a specific encryption status
func (t *Tracker) GetFilesByStatus(scenarioID string, status models.FileStatus) ([]*models.File, error) {
	return t.db.ListFilesByScenario(scenarioID, models.FileFilters{
		Status: status,
	})
}

// GetFilesByDataType returns files of a specific data type
func (t *Tracker) GetFilesByDataType(scenarioID string, dataType string) ([]*models.File, error) {
	return t.db.ListFilesByScenario(scenarioID, models.FileFilters{
		DataType: dataType,
	})
}

// GetStatistics returns file statistics for a scenario
func (t *Tracker) GetStatistics(scenarioID string) (*models.ScenarioStats, error) {
	return t.db.GetScenarioStats(scenarioID)
}

// ProgressCallback is called during bulk operations to report progress
type ProgressCallback func(current, total int, file string)

// TrackDirectoryWithProgress tracks directory with progress reporting
func (t *Tracker) TrackDirectoryWithProgress(scenarioID, root string, dataType string, callback ProgressCallback) (int, error) {
	// First, count total files
	var totalFiles int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalFiles++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	// Now track with progress
	current := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		current++
		if callback != nil {
			callback(current, totalFiles, path)
		}

		if err := t.TrackFile(scenarioID, path, dataType); err != nil {
			return fmt.Errorf("track %s: %w", path, err)
		}

		return nil
	})

	return current, err
}

// FileManifest represents the complete manifest for a scenario
type FileManifest struct {
	ScenarioID string                  `json:"scenario_id"`
	GeneratedAt time.Time              `json:"generated_at"`
	TotalFiles int                     `json:"total_files"`
	TotalSize  int64                   `json:"total_size"`
	Files      []FileManifestEntry     `json:"files"`
	Summary    *models.ScenarioStats   `json:"summary"`
}

// FileManifestEntry represents a single file in the manifest
type FileManifestEntry struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	SHA256    string    `json:"sha256"`
	Size      int64     `json:"size"`
	Extension string    `json:"extension"`
	DataType  string    `json:"data_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// GenerateManifest generates a complete manifest for a scenario
func (t *Tracker) GenerateManifest(scenarioID string) (*FileManifest, error) {
	files, err := t.db.ListFilesByScenario(scenarioID, models.FileFilters{})
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	stats, err := t.GetStatistics(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("get statistics: %w", err)
	}

	manifest := &FileManifest{
		ScenarioID:  scenarioID,
		GeneratedAt: time.Now(),
		TotalFiles:  len(files),
		Files:       make([]FileManifestEntry, len(files)),
		Summary:     stats,
	}

	var totalSize int64
	for i, file := range files {
		manifest.Files[i] = FileManifestEntry{
			ID:        file.ID,
			Path:      file.Path,
			SHA256:    file.SHA256,
			Size:      file.Size,
			Extension: file.Extension,
			DataType:  file.DataType,
			Status:    string(file.EncryptionStatus),
			CreatedAt: file.CreatedAt,
		}
		totalSize += file.Size
	}
	manifest.TotalSize = totalSize

	return manifest, nil
}

// BulkUpdateStatus updates status for multiple files
func (t *Tracker) BulkUpdateStatus(fileIDs []int64, status models.FileStatus) error {
	for _, id := range fileIDs {
		if err := t.UpdateFileStatus(id, status); err != nil {
			return fmt.Errorf("update file %d: %w", id, err)
		}
	}
	return nil
}
