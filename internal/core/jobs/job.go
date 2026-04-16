package jobs

import (
	"encoding/json"
	"time"
)

// JobStatus represents the lifecycle state of a job run.
type JobStatus string

const (
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

// JobRun tracks a single generation run for replay, simulation, and audit.
type JobRun struct {
	ID         string          // Unique run ID (see GenerateRunID)
	ProfileID  string          // Profile that triggered the run, empty for direct generation
	Config     json.RawMessage // Snapshot of generation options at run time
	StartTime  time.Time
	EndTime    *time.Time // nil until the run finishes
	Status     JobStatus
	OutputPath string // Directory where generated files live
	FileCount  int    // Files created (updated on completion)
	ErrorLog   string // Non-empty on failure
	CreatedAt  time.Time
}

// JobFilters narrows a List query.
type JobFilters struct {
	ProfileID string
	Status    JobStatus
	Limit     int
	Offset    int
}

// Store is the persistence interface for job runs.
type Store interface {
	Create(job *JobRun) error
	Get(id string) (*JobRun, error)
	List(filters JobFilters) ([]*JobRun, error)
	Update(job *JobRun) error
	Delete(id string) error
}
