package jobs

import (
	"encoding/json"
	"fmt"
	"time"

	"janus/internal/database/sqlite"
)

// SQLiteStore implements Store using the SQLite database.
type SQLiteStore struct {
	db *sqlite.SQLiteDB
}

// NewSQLiteStore creates a new SQLite-backed job run store.
func NewSQLiteStore(db *sqlite.SQLiteDB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Create(job *JobRun) error {
	configJSON := ""
	if len(job.Config) > 0 {
		configJSON = string(job.Config)
	}
	row := sqlite.JobRunRow{
		ID:         job.ID,
		ProfileID:  job.ProfileID,
		ConfigJSON: configJSON,
		StartTime:  job.StartTime,
		EndTime:    job.EndTime,
		Status:     string(job.Status),
		OutputPath: job.OutputPath,
		FileCount:  job.FileCount,
		ErrorLog:   job.ErrorLog,
		CreatedAt:  job.CreatedAt,
	}
	return s.db.CreateJobRun(row)
}

func (s *SQLiteStore) Get(id string) (*JobRun, error) {
	row, err := s.db.GetJobRun(id)
	if err != nil {
		return nil, err
	}
	return rowToJobRun(row)
}

func (s *SQLiteStore) List(filters JobFilters) ([]*JobRun, error) {
	rows, err := s.db.ListJobRuns(filters.ProfileID, string(filters.Status), filters.Limit, filters.Offset)
	if err != nil {
		return nil, err
	}
	result := make([]*JobRun, 0, len(rows))
	for _, row := range rows {
		job, err := rowToJobRun(row)
		if err != nil {
			return nil, err
		}
		result = append(result, job)
	}
	return result, nil
}

func (s *SQLiteStore) Update(job *JobRun) error {
	configJSON := ""
	if len(job.Config) > 0 {
		configJSON = string(job.Config)
	}
	row := sqlite.JobRunRow{
		ID:         job.ID,
		ProfileID:  job.ProfileID,
		ConfigJSON: configJSON,
		StartTime:  job.StartTime,
		EndTime:    job.EndTime,
		Status:     string(job.Status),
		OutputPath: job.OutputPath,
		FileCount:  job.FileCount,
		ErrorLog:   job.ErrorLog,
		CreatedAt:  job.CreatedAt,
	}
	return s.db.UpdateJobRun(row)
}

func (s *SQLiteStore) Delete(id string) error {
	return s.db.DeleteJobRun(id)
}

func rowToJobRun(row sqlite.JobRunRow) (*JobRun, error) {
	job := &JobRun{
		ID:         row.ID,
		ProfileID:  row.ProfileID,
		StartTime:  row.StartTime,
		EndTime:    row.EndTime,
		Status:     JobStatus(row.Status),
		OutputPath: row.OutputPath,
		FileCount:  row.FileCount,
		ErrorLog:   row.ErrorLog,
		CreatedAt:  row.CreatedAt,
	}
	if row.ConfigJSON != "" {
		job.Config = json.RawMessage(row.ConfigJSON)
	}
	return job, nil
}

// now is a package-level variable to allow overriding in tests.
var now = func() time.Time { return time.Now() }

// StartRun creates a new job run record in the "running" state.
func StartRun(store Store, profileID string, config json.RawMessage, outputPath string) (*JobRun, error) {
	job := &JobRun{
		ID:         GenerateRunID(),
		ProfileID:  profileID,
		Config:     config,
		StartTime:  now(),
		Status:     JobRunning,
		OutputPath: outputPath,
		CreatedAt:  now(),
	}
	if err := store.Create(job); err != nil {
		return nil, fmt.Errorf("create job run: %w", err)
	}
	return job, nil
}

// FinishRun updates a job run to a terminal state.
func FinishRun(store Store, job *JobRun, status JobStatus, fileCount int, errMsg string) error {
	end := now()
	job.EndTime = &end
	job.Status = status
	job.FileCount = fileCount
	job.ErrorLog = errMsg
	return store.Update(job)
}
