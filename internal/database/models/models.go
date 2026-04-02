// Package models defines the data models and database interface for Janus.
package models

import (
	"database/sql"
	"time"
)

// Database is the interface that all database implementations must satisfy.
// This allows swapping between SQLite (standalone) and PostgreSQL (distributed).
type Database interface {
	// Lifecycle
	Connect() error
	Close() error
	Migrate() error
	Ping() error

	// Scenarios
	CreateScenario(s *Scenario) error
	GetScenario(id string) (*Scenario, error)
	ListScenarios() ([]*Scenario, error)
	UpdateScenario(s *Scenario) error
	DeleteScenario(id string) error

	// Files
	CreateFile(f *File) error
	BatchCreateFiles(files []*File) error
	BatchUpdateFileStatus(updates []FileStatusUpdate) error
	GetFile(id int64) (*File, error)
	ListFilesByScenario(scenarioID string, filters FileFilters) ([]*File, error)
	UpdateFile(f *File) error
	DeleteFile(id int64) error
	CountFiles(scenarioID string, status FileStatus) (int, error)

	// Jobs
	CreateJob(j *Job) error
	GetJob(id int64) (*Job, error)
	ListJobs(filters JobFilters) ([]*Job, error)
	UpdateJob(j *Job) error
	DeleteJob(id int64) error
	GetPendingJobs() ([]*Job, error)

	// Tasks (for distributed mode)
	CreateTask(t *Task) error
	GetTask(id int64) (*Task, error)
	ListTasks(filters TaskFilters) ([]*Task, error)
	UpdateTask(t *Task) error
	GetPendingTasksForAgent(agentID string) ([]*Task, error)

	// Agents (for distributed mode)
	RegisterAgent(a *Agent) error
	UpdateAgent(a *Agent) error
	GetAgent(id string) (*Agent, error)
	ListAgents() ([]*Agent, error)
	DeleteAgent(id string) error
	UpdateAgentHeartbeat(id string) error

	// Activity Log
	LogActivity(log *ActivityLog) error
	GetActivityLogs(filters ActivityFilters) ([]*ActivityLog, error)

	// Statistics
	GetScenarioStats(scenarioID string) (*ScenarioStats, error)
}

// Scenario represents a test scenario
type Scenario struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Type        string    `json:"type" db:"type"` // local, distributed
	Config      string    `json:"config" db:"config"` // JSON config
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// File represents a generated file
type File struct {
	ID               int64      `json:"id" db:"id"`
	ScenarioID       string     `json:"scenario_id" db:"scenario_id"`
	Path             string     `json:"path" db:"path"`
	SHA256           string     `json:"sha256" db:"sha256"`
	Size             int64      `json:"size" db:"size"`
	Extension        string     `json:"extension" db:"extension"`
	DataType         string     `json:"data_type" db:"data_type"` // pii, healthcare, financial
	EncryptionStatus FileStatus `json:"encryption_status" db:"encryption_status"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	EncryptedAt      *time.Time `json:"encrypted_at,omitempty" db:"encrypted_at"`
}

// FileStatus represents the encryption status of a file
type FileStatus string

const (
	FileStatusPending   FileStatus = "pending"
	FileStatusEncrypted FileStatus = "encrypted"
	FileStatusFailed    FileStatus = "failed"
)

// FileStatusUpdate carries the fields needed for a batch status update.
type FileStatusUpdate struct {
	FileID      int64
	Status      FileStatus
	EncryptedAt *time.Time // set when Status == FileStatusEncrypted
}

// FileFilters for querying files
type FileFilters struct {
	Status     FileStatus
	DataType   string
	Extension  string
	Limit      int
	Offset     int
}

// Job represents a scheduled encryption job
type Job struct {
	ID               int64      `json:"id" db:"id"`
	ScenarioID       string     `json:"scenario_id" db:"scenario_id"`
	ScheduledAt      time.Time  `json:"scheduled_at" db:"scheduled_at"`
	ExecutedAt       *time.Time `json:"executed_at,omitempty" db:"executed_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	TargetPercentage float64    `json:"target_percentage" db:"target_percentage"`
	FilesEncrypted   int        `json:"files_encrypted" db:"files_encrypted"`
	Status           JobStatus  `json:"status" db:"status"`
	Error            string     `json:"error,omitempty" db:"error"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// JobFilters for querying jobs
type JobFilters struct {
	ScenarioID string
	Status     JobStatus
	Limit      int
	Offset     int
}

// Task represents a task for an agent (distributed mode)
type Task struct {
	ID         int64      `json:"id" db:"id"`
	ScenarioID string     `json:"scenario_id" db:"scenario_id"`
	AgentID    string     `json:"agent_id" db:"agent_id"`
	Action     string     `json:"action" db:"action"` // generate, encrypt, destroy
	Config     string     `json:"config" db:"config"` // JSON config
	Status     TaskStatus `json:"status" db:"status"`
	Result     string     `json:"result,omitempty" db:"result"`
	Error      string     `json:"error,omitempty" db:"error"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// TaskFilters for querying tasks
type TaskFilters struct {
	ScenarioID string
	AgentID    string
	Status     TaskStatus
	Limit      int
	Offset     int
}

// Agent represents a remote agent (distributed mode)
type Agent struct {
	ID               string    `json:"id" db:"id"`
	Hostname         string    `json:"hostname" db:"hostname"`
	OS               string    `json:"os" db:"os"`
	Arch             string    `json:"arch" db:"arch"`
	Version          string    `json:"version" db:"version"`
	Status           string    `json:"status" db:"status"` // online, offline
	LastHeartbeat    time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	RegisteredAt     time.Time `json:"registered_at" db:"registered_at"`
	TasksCompleted   int       `json:"tasks_completed" db:"tasks_completed"`
	TasksFailed      int       `json:"tasks_failed" db:"tasks_failed"`
}

// ActivityLog represents an activity log entry
type ActivityLog struct {
	ID        int64     `json:"id" db:"id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Level     string    `json:"level" db:"level"` // info, warning, error
	Action    string    `json:"action" db:"action"` // generate, encrypt, delete, etc.
	FileID    *int64    `json:"file_id,omitempty" db:"file_id"`
	Details   string    `json:"details" db:"details"`
	AgentID   *string   `json:"agent_id,omitempty" db:"agent_id"`
}

// ActivityFilters for querying activity logs
type ActivityFilters struct {
	Level      string
	Action     string
	ScenarioID string
	Limit      int
	Offset     int
}

// ScenarioStats represents statistics for a scenario
type ScenarioStats struct {
	ScenarioID      string  `json:"scenario_id"`
	TotalFiles      int     `json:"total_files"`
	EncryptedFiles  int     `json:"encrypted_files"`
	PendingFiles    int     `json:"pending_files"`
	FailedFiles     int     `json:"failed_files"`
	TotalSize       int64   `json:"total_size"`
	EncryptedSize   int64   `json:"encrypted_size"`
	EncryptedPercent float64 `json:"encrypted_percent"`
	ByDataType      map[string]int `json:"by_data_type"`
	ByExtension     map[string]int `json:"by_extension"`
}

// NullTime represents a nullable time
type NullTime struct {
	sql.NullTime
}

// MarshalJSON implements json.Marshaler
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return nt.Time.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler
func (nt *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}
	if err := nt.Time.UnmarshalJSON(data); err != nil {
		return err
	}
	nt.Valid = true
	return nil
}
