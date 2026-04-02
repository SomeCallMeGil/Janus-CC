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

// WriteCSV writes the file manifest for scenarioID as CSV to w.
// Callers are responsible for setting Content-Type and Content-Disposition headers.
func (t *Tracker) WriteCSV(scenarioID string, w io.Writer) error {
	files, err := t.db.ListFilesByScenario(scenarioID, models.FileFilters{})
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"ID", "Path", "SHA256", "Size", "Extension", "DataType", "EncryptionStatus", "CreatedAt", "EncryptedAt"}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

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
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

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

