// Package scheduler provides job scheduling functionality.
package scheduler

import (
	"fmt"
	"math/rand"
	"time"

	"janus/internal/database/models"
)

// Scheduler handles job scheduling and execution
type Scheduler struct {
	db models.Database
}

// New creates a new scheduler
func New(db models.Database) *Scheduler {
	return &Scheduler{db: db}
}

// ScheduleJob schedules a new encryption job
func (s *Scheduler) ScheduleJob(scenarioID string, scheduledAt time.Time, targetPercentage float64) (*models.Job, error) {
	if targetPercentage < 0 || targetPercentage > 100 {
		return nil, fmt.Errorf("target percentage must be 0-100, got %.2f", targetPercentage)
	}

	job := &models.Job{
		ScenarioID:       scenarioID,
		ScheduledAt:      scheduledAt,
		TargetPercentage: targetPercentage,
		Status:           models.JobStatusPending,
		CreatedAt:        time.Now(),
	}

	if err := s.db.CreateJob(job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	return job, nil
}

// ScheduleIncremental creates multiple jobs for incremental encryption
func (s *Scheduler) ScheduleIncremental(scenarioID string, startDate time.Time, days int, totalPercentage float64) ([]*models.Job, error) {
	if days < 1 {
		return nil, fmt.Errorf("days must be at least 1")
	}

	dailyPercentage := totalPercentage / float64(days)
	var jobs []*models.Job

	for i := 0; i < days; i++ {
		scheduledAt := startDate.AddDate(0, 0, i)
		
		job, err := s.ScheduleJob(scenarioID, scheduledAt, dailyPercentage)
		if err != nil {
			return jobs, fmt.Errorf("schedule job day %d: %w", i+1, err)
		}
		
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetPendingJobs returns jobs ready to be executed
func (s *Scheduler) GetPendingJobs() ([]*models.Job, error) {
	return s.db.GetPendingJobs()
}

// GetJob returns a job by ID
func (s *Scheduler) GetJob(id int64) (*models.Job, error) {
	return s.db.GetJob(id)
}

// ListJobs lists jobs with filters
func (s *Scheduler) ListJobs(scenarioID string, status models.JobStatus) ([]*models.Job, error) {
	filters := models.JobFilters{
		ScenarioID: scenarioID,
		Status:     status,
	}
	return s.db.ListJobs(filters)
}

// UpdateJobStatus updates the status of a job
func (s *Scheduler) UpdateJobStatus(id int64, status models.JobStatus) error {
	job, err := s.db.GetJob(id)
	if err != nil {
		return err
	}

	job.Status = status
	
	switch status {
	case models.JobStatusRunning:
		now := time.Now()
		job.ExecutedAt = &now
	case models.JobStatusCompleted, models.JobStatusFailed, models.JobStatusCancelled:
		now := time.Now()
		job.CompletedAt = &now
	}

	return s.db.UpdateJob(job)
}

// UpdateJobProgress updates the progress of a running job
func (s *Scheduler) UpdateJobProgress(id int64, filesEncrypted int) error {
	job, err := s.db.GetJob(id)
	if err != nil {
		return err
	}

	job.FilesEncrypted = filesEncrypted
	return s.db.UpdateJob(job)
}

// CancelJob cancels a pending job
func (s *Scheduler) CancelJob(id int64) error {
	job, err := s.db.GetJob(id)
	if err != nil {
		return err
	}

	if job.Status != models.JobStatusPending {
		return fmt.Errorf("can only cancel pending jobs, job is %s", job.Status)
	}

	job.Status = models.JobStatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	return s.db.UpdateJob(job)
}

// SelectFilesForJob selects files to encrypt for a job
func (s *Scheduler) SelectFilesForJob(scenarioID string, targetPercentage float64, method string) ([]int64, error) {
	// Get total file count
	totalFiles, err := s.db.CountFiles(scenarioID, "")
	if err != nil {
		return nil, fmt.Errorf("count files: %w", err)
	}

	// Get already encrypted files
	encryptedCount, err := s.db.CountFiles(scenarioID, models.FileStatusEncrypted)
	if err != nil {
		return nil, fmt.Errorf("count encrypted: %w", err)
	}

	// Calculate target count
	targetCount := int(float64(totalFiles) * targetPercentage / 100.0)
	
	// Subtract already encrypted
	remainingToEncrypt := targetCount - encryptedCount
	if remainingToEncrypt <= 0 {
		return []int64{}, nil // Already met target
	}

	// Get pending files
	pendingFiles, err := s.db.ListFilesByScenario(scenarioID, models.FileFilters{
		Status: models.FileStatusPending,
	})
	if err != nil {
		return nil, fmt.Errorf("list pending files: %w", err)
	}

	// Select files based on method
	var selectedIDs []int64
	
	switch method {
	case "random":
		selectedIDs = s.selectRandom(pendingFiles, remainingToEncrypt)
	case "sequential":
		selectedIDs = s.selectSequential(pendingFiles, remainingToEncrypt)
	case "largest":
		selectedIDs = s.selectLargest(pendingFiles, remainingToEncrypt)
	case "smallest":
		selectedIDs = s.selectSmallest(pendingFiles, remainingToEncrypt)
	default:
		selectedIDs = s.selectRandom(pendingFiles, remainingToEncrypt)
	}

	return selectedIDs, nil
}

// selectRandom randomly selects files
func (s *Scheduler) selectRandom(files []*models.File, count int) []int64 {
	if count > len(files) {
		count = len(files)
	}

	// Shuffle files
	rand.Shuffle(len(files), func(i, j int) {
		files[i], files[j] = files[j], files[i]
	})

	ids := make([]int64, count)
	for i := 0; i < count; i++ {
		ids[i] = files[i].ID
	}
	return ids
}

// selectSequential selects files in order
func (s *Scheduler) selectSequential(files []*models.File, count int) []int64 {
	if count > len(files) {
		count = len(files)
	}

	ids := make([]int64, count)
	for i := 0; i < count; i++ {
		ids[i] = files[i].ID
	}
	return ids
}

// selectLargest selects largest files first
func (s *Scheduler) selectLargest(files []*models.File, count int) []int64 {
	if count > len(files) {
		count = len(files)
	}

	// Sort by size descending
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].Size > files[i].Size {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	ids := make([]int64, count)
	for i := 0; i < count; i++ {
		ids[i] = files[i].ID
	}
	return ids
}

// selectSmallest selects smallest files first
func (s *Scheduler) selectSmallest(files []*models.File, count int) []int64 {
	if count > len(files) {
		count = len(files)
	}

	// Sort by size ascending
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].Size < files[i].Size {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	ids := make([]int64, count)
	for i := 0; i < count; i++ {
		ids[i] = files[i].ID
	}
	return ids
}

// GetJobProgress calculates job progress
func (s *Scheduler) GetJobProgress(id int64) (*JobProgress, error) {
	job, err := s.db.GetJob(id)
	if err != nil {
		return nil, err
	}

	// Get file counts
	totalFiles, _ := s.db.CountFiles(job.ScenarioID, "")
	targetCount := int(float64(totalFiles) * job.TargetPercentage / 100.0)
	
	progress := &JobProgress{
		JobID:           job.ID,
		Status:          job.Status,
		TargetCount:     targetCount,
		EncryptedCount:  job.FilesEncrypted,
		RemainingCount:  targetCount - job.FilesEncrypted,
		PercentComplete: 0,
	}

	if targetCount > 0 {
		progress.PercentComplete = float64(job.FilesEncrypted) / float64(targetCount) * 100.0
	}

	if job.ExecutedAt != nil {
		progress.Duration = time.Since(*job.ExecutedAt)
	}

	return progress, nil
}

// JobProgress represents job execution progress
type JobProgress struct {
	JobID           int64           `json:"job_id"`
	Status          models.JobStatus `json:"status"`
	TargetCount     int             `json:"target_count"`
	EncryptedCount  int             `json:"encrypted_count"`
	RemainingCount  int             `json:"remaining_count"`
	PercentComplete float64         `json:"percent_complete"`
	Duration        time.Duration   `json:"duration"`
}

// Assessment provides scenario assessment
type Assessment struct {
	ScenarioID       string  `json:"scenario_id"`
	TotalFiles       int     `json:"total_files"`
	TotalSize        int64   `json:"total_size"`
	EncryptedFiles   int     `json:"encrypted_files"`
	PendingFiles     int     `json:"pending_files"`
	EncryptedPercent float64 `json:"encrypted_percent"`
	ByDataType       map[string]int `json:"by_data_type"`
	ByExtension      map[string]int `json:"by_extension"`
}

// AssessScenario assesses a scenario's current state
func (s *Scheduler) AssessScenario(scenarioID string) (*Assessment, error) {
	stats, err := s.db.GetScenarioStats(scenarioID)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	assessment := &Assessment{
		ScenarioID:       scenarioID,
		TotalFiles:       stats.TotalFiles,
		TotalSize:        stats.TotalSize,
		EncryptedFiles:   stats.EncryptedFiles,
		PendingFiles:     stats.PendingFiles,
		EncryptedPercent: stats.EncryptedPercent,
		ByDataType:       stats.ByDataType,
		ByExtension:      stats.ByExtension,
	}

	return assessment, nil
}
