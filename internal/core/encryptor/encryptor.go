// Package encryptor orchestrates file encryption operations.
package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/pbkdf2"
	"janus/internal/core/scheduler"
	"janus/internal/core/tracker"
	"janus/internal/database/models"
)

const (
	// HeaderMagic identifies encrypted files
	HeaderMagic = "JNS2"
	
	// HeaderSize is the total header size
	HeaderSize = 60
	
	// Salt, nonce, and tag sizes for AES-GCM
	SaltSize  = 16
	NonceSize = 12
	TagSize   = 16
	
	// Encryption modes
	ModeFullEncryption    = 0
	ModePartialEncryption = 1
	
	// Default values
	DefaultPBKDFIterations = 100000
	MinPBKDFIterations     = 10000
	MaxPartialBytes        = 100 * 1024 * 1024 // 100MB
	BufferSize             = 64 * 1024          // 64KB
)

// Encryptor handles file encryption operations
type Encryptor struct {
	db        models.Database
	tracker   *tracker.Tracker
	scheduler *scheduler.Scheduler
	password  []byte
	workers   int
}

// New creates a new encryptor
func New(db models.Database, password string, workers int) *Encryptor {
	if workers < 1 {
		workers = 1
	}
	
	return &Encryptor{
		db:        db,
		tracker:   tracker.New(db),
		scheduler: scheduler.New(db),
		password:  []byte(password),
		workers:   workers,
	}
}

// Options for encryption
type Options struct {
	Mode            int   // Full or partial
	PartialBytes    int64 // For partial encryption
	PBKDFIterations int
	Workers         int
}

// DefaultOptions returns default encryption options
func DefaultOptions() *Options {
	return &Options{
		Mode:            ModePartialEncryption,
		PartialBytes:    4096,
		PBKDFIterations: DefaultPBKDFIterations,
		Workers:         4,
	}
}

// EncryptFile encrypts a single file
func (e *Encryptor) EncryptFile(filePath string, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}
	
	// Validate options
	if err := e.validateOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}
	
	// Generate salt
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	
	// Derive key from password
	key := pbkdf2.Key(e.password, salt, opts.PBKDFIterations, 32, sha256.New)
	
	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}
	
	// Generate nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}
	
	// Open source file
	src, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()
	
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	
	// Determine bytes to encrypt
	var bytesToEncrypt int64
	if opts.Mode == ModePartialEncryption {
		bytesToEncrypt = opts.PartialBytes
		if bytesToEncrypt > info.Size() {
			bytesToEncrypt = info.Size()
		}
	} else {
		bytesToEncrypt = info.Size()
	}
	
	// Create temporary output file
	dir := filepath.Dir(filePath)
	tmpDst, err := os.CreateTemp(dir, ".janus-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpDst.Name()
	defer os.Remove(tmpPath) // Clean up on error
	
	// Write header
	header := e.createHeader(opts.Mode, bytesToEncrypt, salt, nonce)
	if _, err := tmpDst.Write(header); err != nil {
		tmpDst.Close()
		return fmt.Errorf("write header: %w", err)
	}
	
	// Read and encrypt data
	plaintext := make([]byte, bytesToEncrypt)
	n, err := io.ReadFull(src, plaintext)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		tmpDst.Close()
		return fmt.Errorf("read data: %w", err)
	}
	plaintext = plaintext[:n]
	
	// Encrypt with GCM (includes authentication tag)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	
	// Write encrypted data
	if _, err := tmpDst.Write(ciphertext); err != nil {
		tmpDst.Close()
		return fmt.Errorf("write ciphertext: %w", err)
	}
	
	// For partial encryption, copy remaining bytes
	if opts.Mode == ModePartialEncryption && bytesToEncrypt < info.Size() {
		if _, err := io.Copy(tmpDst, src); err != nil {
			tmpDst.Close()
			return fmt.Errorf("copy remaining: %w", err)
		}
	}
	
	// Sync to disk
	if err := tmpDst.Sync(); err != nil {
		tmpDst.Close()
		return fmt.Errorf("sync: %w", err)
	}
	tmpDst.Close()
	
	// Atomic rename (overwrites original)
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	
	return nil
}

// createHeader creates the file header
func (e *Encryptor) createHeader(mode int, partialBytes int64, salt, nonce []byte) []byte {
	header := make([]byte, HeaderSize)
	
	// Magic (4 bytes)
	copy(header[0:4], []byte(HeaderMagic))
	
	// Mode (1 byte)
	header[4] = byte(mode)
	
	// Padding (3 bytes) - reserved
	
	// Partial bytes (8 bytes)
	for i := 0; i < 8; i++ {
		header[8+i] = byte(partialBytes >> (i * 8))
	}
	
	// Salt (16 bytes)
	copy(header[16:32], salt)
	
	// Nonce (12 bytes)
	copy(header[32:44], nonce)
	
	// Tag will be appended by GCM after ciphertext
	
	return header
}

// validateOptions validates encryption options
func (e *Encryptor) validateOptions(opts *Options) error {
	if opts.Mode != ModeFullEncryption && opts.Mode != ModePartialEncryption {
		return fmt.Errorf("invalid mode: %d", opts.Mode)
	}
	
	if opts.Mode == ModePartialEncryption {
		if opts.PartialBytes <= 0 {
			return fmt.Errorf("partial bytes must be > 0")
		}
		if opts.PartialBytes > MaxPartialBytes {
			return fmt.Errorf("partial bytes exceeds maximum (%d)", MaxPartialBytes)
		}
	}
	
	if opts.PBKDFIterations < MinPBKDFIterations {
		return fmt.Errorf("PBKDF iterations must be >= %d", MinPBKDFIterations)
	}
	
	return nil
}

// EncryptJob encrypts files for a job
func (e *Encryptor) EncryptJob(jobID int64, opts *Options, progress ProgressCallback) error {
	// Get job
	job, err := e.scheduler.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}
	
	// Update job status to running
	if err := e.scheduler.UpdateJobStatus(jobID, models.JobStatusRunning); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	
	// Select files to encrypt
	fileIDs, err := e.scheduler.SelectFilesForJob(job.ScenarioID, job.TargetPercentage, "random")
	if err != nil {
		e.scheduler.UpdateJobStatus(jobID, models.JobStatusFailed)
		return fmt.Errorf("select files: %w", err)
	}
	
	if len(fileIDs) == 0 {
		// No files to encrypt, mark as completed
		e.scheduler.UpdateJobStatus(jobID, models.JobStatusCompleted)
		return nil
	}
	
	// Encrypt files concurrently
	results := e.encryptConcurrent(fileIDs, opts, progress)
	
	// Batch-update all file statuses in a single transaction instead of
	// calling UpdateFileStatus (GET+UPDATE) per file.
	successCount := 0
	updates := make([]models.FileStatusUpdate, 0, len(results))
	for _, result := range results {
		u := models.FileStatusUpdate{FileID: result.FileID, Status: models.FileStatusFailed}
		if result.Error == nil {
			successCount++
			now := time.Now()
			u.Status = models.FileStatusEncrypted
			u.EncryptedAt = &now
		}
		updates = append(updates, u)
	}
	if err := e.db.BatchUpdateFileStatus(updates); err != nil {
		// Log but don't fail the job — encryption already happened on disk.
		log.Warn().Err(err).Int64("job_id", jobID).Msg("batch status update failed")
	}
	
	// Update job progress
	e.scheduler.UpdateJobProgress(jobID, successCount)
	
	// Update job status
	if successCount == len(fileIDs) {
		e.scheduler.UpdateJobStatus(jobID, models.JobStatusCompleted)
	} else if successCount == 0 {
		e.scheduler.UpdateJobStatus(jobID, models.JobStatusFailed)
	} else {
		// Partial success still counts as completed
		e.scheduler.UpdateJobStatus(jobID, models.JobStatusCompleted)
	}
	
	return nil
}

// encryptConcurrent encrypts files concurrently
func (e *Encryptor) encryptConcurrent(fileIDs []int64, opts *Options, progress ProgressCallback) []EncryptResult {
	jobs := make(chan int64, len(fileIDs))
	results := make(chan EncryptResult, len(fileIDs))
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < e.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fileID := range jobs {
				result := e.encryptSingleFile(fileID, opts)
				results <- result
				
				if progress != nil {
					progress(result)
				}
			}
		}()
	}
	
	// Send jobs
	for _, fileID := range fileIDs {
		jobs <- fileID
	}
	close(jobs)
	
	// Wait for completion
	wg.Wait()
	close(results)
	
	// Collect results
	var allResults []EncryptResult
	for result := range results {
		allResults = append(allResults, result)
	}
	
	return allResults
}

// encryptSingleFile encrypts a single file by ID
func (e *Encryptor) encryptSingleFile(fileID int64, opts *Options) EncryptResult {
	file, err := e.db.GetFile(fileID)
	if err != nil {
		return EncryptResult{
			FileID: fileID,
			Error:  fmt.Errorf("get file: %w", err),
		}
	}
	
	startTime := time.Now()
	err = e.EncryptFile(file.Path, opts)
	duration := time.Since(startTime)
	
	return EncryptResult{
		FileID:   fileID,
		FilePath: file.Path,
		Duration: duration,
		Error:    err,
	}
}

// EncryptResult represents the result of encrypting a file
type EncryptResult struct {
	FileID   int64
	FilePath string
	Duration time.Duration
	Error    error
}

// ProgressCallback is called for each file encrypted
type ProgressCallback func(result EncryptResult)

// EncryptScenario encrypts a percentage of files in a scenario
func (e *Encryptor) EncryptScenario(scenarioID string, percentage float64, opts *Options, progress ProgressCallback) error {
	if opts == nil {
		opts = DefaultOptions()
	}
	
	// Create immediate job
	job, err := e.scheduler.ScheduleJob(scenarioID, time.Now(), percentage)
	if err != nil {
		return fmt.Errorf("schedule job: %w", err)
	}
	
	// Execute job
	return e.EncryptJob(job.ID, opts, progress)
}

// GetStats returns encryption statistics
type Stats struct {
	TotalFiles     int
	EncryptedFiles int
	PendingFiles   int
	FailedFiles    int
	Percentage     float64
}

// GetEncryptionStats returns encryption statistics for a scenario
func (e *Encryptor) GetEncryptionStats(scenarioID string) (*Stats, error) {
	stats, err := e.tracker.GetStatistics(scenarioID)
	if err != nil {
		return nil, err
	}
	
	return &Stats{
		TotalFiles:     stats.TotalFiles,
		EncryptedFiles: stats.EncryptedFiles,
		PendingFiles:   stats.PendingFiles,
		FailedFiles:    stats.FailedFiles,
		Percentage:     stats.EncryptedPercent,
	}, nil
}
