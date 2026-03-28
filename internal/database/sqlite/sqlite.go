// Package sqlite provides SQLite database implementation for standalone mode.
package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"janus/internal/database/models"

	_ "modernc.org/sqlite"
)

// SQLiteDB implements the Database interface using SQLite
type SQLiteDB struct {
	db   *sql.DB
	path string
}

// New creates a new SQLite database connection
func New(path string) (*SQLiteDB, error) {
	return &SQLiteDB{
		path: path,
	}, nil
}

// Connect opens the database connection
func (s *SQLiteDB) Connect() error {
	db, err := sql.Open("sqlite", s.path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	s.db = db
	return nil
}

// Close closes the database connection
func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping checks if the database is accessible
func (s *SQLiteDB) Ping() error {
	return s.db.Ping()
}

// Migrate creates or updates the database schema
func (s *SQLiteDB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS scenarios (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		type TEXT NOT NULL DEFAULT 'local',
		config TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scenario_id TEXT NOT NULL,
		path TEXT NOT NULL,
		sha256 TEXT NOT NULL,
		size INTEGER NOT NULL,
		extension TEXT,
		data_type TEXT,
		encryption_status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		encrypted_at TIMESTAMP,
		FOREIGN KEY (scenario_id) REFERENCES scenarios(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scenario_id TEXT NOT NULL,
		scheduled_at TIMESTAMP NOT NULL,
		executed_at TIMESTAMP,
		completed_at TIMESTAMP,
		target_percentage REAL NOT NULL,
		files_encrypted INTEGER DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'pending',
		error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (scenario_id) REFERENCES scenarios(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scenario_id TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		action TEXT NOT NULL,
		config TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		result TEXT,
		error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		completed_at TIMESTAMP,
		FOREIGN KEY (scenario_id) REFERENCES scenarios(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		hostname TEXT NOT NULL,
		os TEXT NOT NULL,
		arch TEXT NOT NULL,
		version TEXT,
		status TEXT NOT NULL DEFAULT 'online',
		last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		tasks_completed INTEGER DEFAULT 0,
		tasks_failed INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS activity_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		level TEXT NOT NULL,
		action TEXT NOT NULL,
		file_id INTEGER,
		details TEXT,
		agent_id TEXT,
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_files_scenario ON files(scenario_id);
	CREATE INDEX IF NOT EXISTS idx_files_status ON files(encryption_status);
	CREATE INDEX IF NOT EXISTS idx_jobs_scenario ON jobs(scenario_id);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_agent ON tasks(agent_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_activity_timestamp ON activity_log(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_activity_level ON activity_log(level);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateScenario creates a new scenario
func (s *SQLiteDB) CreateScenario(scenario *models.Scenario) error {
	query := `
		INSERT INTO scenarios (id, name, description, type, config, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		scenario.ID,
		scenario.Name,
		scenario.Description,
		scenario.Type,
		scenario.Config,
		scenario.Status,
		scenario.CreatedAt,
		scenario.UpdatedAt,
	)
	return err
}

// GetScenario retrieves a scenario by ID
func (s *SQLiteDB) GetScenario(id string) (*models.Scenario, error) {
	query := `
		SELECT id, name, description, type, config, status, created_at, updated_at
		FROM scenarios WHERE id = ?
	`
	scenario := &models.Scenario{}
	err := s.db.QueryRow(query, id).Scan(
		&scenario.ID,
		&scenario.Name,
		&scenario.Description,
		&scenario.Type,
		&scenario.Config,
		&scenario.Status,
		&scenario.CreatedAt,
		&scenario.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scenario not found: %s", id)
	}
	return scenario, err
}

// ListScenarios returns all scenarios
func (s *SQLiteDB) ListScenarios() ([]*models.Scenario, error) {
	query := `
		SELECT id, name, description, type, config, status, created_at, updated_at
		FROM scenarios ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []*models.Scenario
	for rows.Next() {
		scenario := &models.Scenario{}
		err := rows.Scan(
			&scenario.ID,
			&scenario.Name,
			&scenario.Description,
			&scenario.Type,
			&scenario.Config,
			&scenario.Status,
			&scenario.CreatedAt,
			&scenario.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios, rows.Err()
}

// UpdateScenario updates a scenario
func (s *SQLiteDB) UpdateScenario(scenario *models.Scenario) error {
	query := `
		UPDATE scenarios 
		SET name = ?, description = ?, type = ?, config = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	scenario.UpdatedAt = time.Now()
	_, err := s.db.Exec(query,
		scenario.Name,
		scenario.Description,
		scenario.Type,
		scenario.Config,
		scenario.Status,
		scenario.UpdatedAt,
		scenario.ID,
	)
	return err
}

// DeleteScenario deletes a scenario
func (s *SQLiteDB) DeleteScenario(id string) error {
	_, err := s.db.Exec("DELETE FROM scenarios WHERE id = ?", id)
	return err
}

// CreateFile creates a new file record
func (s *SQLiteDB) CreateFile(file *models.File) error {
	query := `
		INSERT INTO files (scenario_id, path, sha256, size, extension, data_type, encryption_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(query,
		file.ScenarioID,
		file.Path,
		file.SHA256,
		file.Size,
		file.Extension,
		file.DataType,
		file.EncryptionStatus,
		file.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	file.ID = id
	return nil
}

// GetFile retrieves a file by ID
func (s *SQLiteDB) GetFile(id int64) (*models.File, error) {
	query := `
		SELECT id, scenario_id, path, sha256, size, extension, data_type, 
		       encryption_status, created_at, encrypted_at
		FROM files WHERE id = ?
	`
	file := &models.File{}
	var encryptedAt sql.NullTime
	err := s.db.QueryRow(query, id).Scan(
		&file.ID,
		&file.ScenarioID,
		&file.Path,
		&file.SHA256,
		&file.Size,
		&file.Extension,
		&file.DataType,
		&file.EncryptionStatus,
		&file.CreatedAt,
		&encryptedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found: %d", id)
	}
	if encryptedAt.Valid {
		file.EncryptedAt = &encryptedAt.Time
	}
	return file, err
}

// ListFilesByScenario returns files for a scenario with filters
func (s *SQLiteDB) ListFilesByScenario(scenarioID string, filters models.FileFilters) ([]*models.File, error) {
	query := `
		SELECT id, scenario_id, path, sha256, size, extension, data_type,
		       encryption_status, created_at, encrypted_at
		FROM files WHERE scenario_id = ?
	`
	args := []interface{}{scenarioID}

	if filters.Status != "" {
		query += " AND encryption_status = ?"
		args = append(args, filters.Status)
	}
	if filters.DataType != "" {
		query += " AND data_type = ?"
		args = append(args, filters.DataType)
	}
	if filters.Extension != "" {
		query += " AND extension = ?"
		args = append(args, filters.Extension)
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.File
	for rows.Next() {
		file := &models.File{}
		var encryptedAt sql.NullTime
		err := rows.Scan(
			&file.ID,
			&file.ScenarioID,
			&file.Path,
			&file.SHA256,
			&file.Size,
			&file.Extension,
			&file.DataType,
			&file.EncryptionStatus,
			&file.CreatedAt,
			&encryptedAt,
		)
		if err != nil {
			return nil, err
		}
		if encryptedAt.Valid {
			file.EncryptedAt = &encryptedAt.Time
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

// UpdateFile updates a file record
func (s *SQLiteDB) UpdateFile(file *models.File) error {
	query := `
		UPDATE files 
		SET path = ?, sha256 = ?, size = ?, extension = ?, data_type = ?,
		    encryption_status = ?, encrypted_at = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query,
		file.Path,
		file.SHA256,
		file.Size,
		file.Extension,
		file.DataType,
		file.EncryptionStatus,
		file.EncryptedAt,
		file.ID,
	)
	return err
}

// DeleteFile deletes a file record
func (s *SQLiteDB) DeleteFile(id int64) error {
	_, err := s.db.Exec("DELETE FROM files WHERE id = ?", id)
	return err
}

// CountFiles counts files by status
func (s *SQLiteDB) CountFiles(scenarioID string, status models.FileStatus) (int, error) {
	query := "SELECT COUNT(*) FROM files WHERE scenario_id = ?"
	args := []interface{}{scenarioID}

	if status != "" {
		query += " AND encryption_status = ?"
		args = append(args, status)
	}

	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// CreateJob creates a new job
func (s *SQLiteDB) CreateJob(job *models.Job) error {
	query := `
		INSERT INTO jobs (scenario_id, scheduled_at, target_percentage, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(query,
		job.ScenarioID,
		job.ScheduledAt,
		job.TargetPercentage,
		job.Status,
		job.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	job.ID = id
	return nil
}

// GetJob retrieves a job by ID
func (s *SQLiteDB) GetJob(id int64) (*models.Job, error) {
	query := `
		SELECT id, scenario_id, scheduled_at, executed_at, completed_at, 
		       target_percentage, files_encrypted, status, error, created_at
		FROM jobs WHERE id = ?
	`
	job := &models.Job{}
	var executedAt, completedAt sql.NullTime
	var errorMsg sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&job.ID,
		&job.ScenarioID,
		&job.ScheduledAt,
		&executedAt,
		&completedAt,
		&job.TargetPercentage,
		&job.FilesEncrypted,
		&job.Status,
		&errorMsg,
		&job.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %d", id)
	}
	if executedAt.Valid {
		job.ExecutedAt = &executedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}
	return job, err
}

// ListJobs returns jobs with filters
func (s *SQLiteDB) ListJobs(filters models.JobFilters) ([]*models.Job, error) {
	query := `
		SELECT id, scenario_id, scheduled_at, executed_at, completed_at,
		       target_percentage, files_encrypted, status, error, created_at
		FROM jobs WHERE 1=1
	`
	args := []interface{}{}

	if filters.ScenarioID != "" {
		query += " AND scenario_id = ?"
		args = append(args, filters.ScenarioID)
	}
	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}

	query += " ORDER BY scheduled_at DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		var executedAt, completedAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&job.ID,
			&job.ScenarioID,
			&job.ScheduledAt,
			&executedAt,
			&completedAt,
			&job.TargetPercentage,
			&job.FilesEncrypted,
			&job.Status,
			&errorMsg,
			&job.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if executedAt.Valid {
			job.ExecutedAt = &executedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if errorMsg.Valid {
			job.Error = errorMsg.String
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// UpdateJob updates a job
func (s *SQLiteDB) UpdateJob(job *models.Job) error {
	query := `
		UPDATE jobs 
		SET executed_at = ?, completed_at = ?, files_encrypted = ?, status = ?, error = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query,
		job.ExecutedAt,
		job.CompletedAt,
		job.FilesEncrypted,
		job.Status,
		job.Error,
		job.ID,
	)
	return err
}

// DeleteJob deletes a job
func (s *SQLiteDB) DeleteJob(id int64) error {
	_, err := s.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	return err
}

// GetPendingJobs returns jobs that are ready to execute
func (s *SQLiteDB) GetPendingJobs() ([]*models.Job, error) {
	query := `
		SELECT id, scenario_id, scheduled_at, executed_at, completed_at,
		       target_percentage, files_encrypted, status, error, created_at
		FROM jobs 
		WHERE status = 'pending' AND scheduled_at <= datetime('now')
		ORDER BY scheduled_at ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		var executedAt, completedAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&job.ID,
			&job.ScenarioID,
			&job.ScheduledAt,
			&executedAt,
			&completedAt,
			&job.TargetPercentage,
			&job.FilesEncrypted,
			&job.Status,
			&errorMsg,
			&job.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if executedAt.Valid {
			job.ExecutedAt = &executedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if errorMsg.Valid {
			job.Error = errorMsg.String
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// Task methods (stubs for distributed mode - not used in standalone)
func (s *SQLiteDB) CreateTask(t *models.Task) error                                { return nil }
func (s *SQLiteDB) GetTask(id int64) (*models.Task, error)                         { return nil, nil }
func (s *SQLiteDB) ListTasks(filters models.TaskFilters) ([]*models.Task, error)   { return nil, nil }
func (s *SQLiteDB) UpdateTask(t *models.Task) error                                { return nil }
func (s *SQLiteDB) GetPendingTasksForAgent(agentID string) ([]*models.Task, error) { return nil, nil }

// Agent methods (stubs for distributed mode - not used in standalone)
func (s *SQLiteDB) RegisterAgent(a *models.Agent) error       { return nil }
func (s *SQLiteDB) UpdateAgent(a *models.Agent) error         { return nil }
func (s *SQLiteDB) GetAgent(id string) (*models.Agent, error) { return nil, nil }
func (s *SQLiteDB) ListAgents() ([]*models.Agent, error)      { return nil, nil }
func (s *SQLiteDB) DeleteAgent(id string) error               { return nil }
func (s *SQLiteDB) UpdateAgentHeartbeat(id string) error      { return nil }

// LogActivity logs an activity
func (s *SQLiteDB) LogActivity(log *models.ActivityLog) error {
	query := `
		INSERT INTO activity_log (timestamp, level, action, file_id, details, agent_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		log.Timestamp,
		log.Level,
		log.Action,
		log.FileID,
		log.Details,
		log.AgentID,
	)
	return err
}

// GetActivityLogs returns activity logs with filters
func (s *SQLiteDB) GetActivityLogs(filters models.ActivityFilters) ([]*models.ActivityLog, error) {
	query := `
		SELECT id, timestamp, level, action, file_id, details, agent_id
		FROM activity_log WHERE 1=1
	`
	args := []interface{}{}

	if filters.Level != "" {
		query += " AND level = ?"
		args = append(args, filters.Level)
	}
	if filters.Action != "" {
		query += " AND action = ?"
		args = append(args, filters.Action)
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ActivityLog
	for rows.Next() {
		log := &models.ActivityLog{}
		var fileID sql.NullInt64
		var agentID sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Level,
			&log.Action,
			&fileID,
			&log.Details,
			&agentID,
		)
		if err != nil {
			return nil, err
		}
		if fileID.Valid {
			id := fileID.Int64
			log.FileID = &id
		}
		if agentID.Valid {
			id := agentID.String
			log.AgentID = &id
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// GetScenarioStats returns statistics for a scenario
func (s *SQLiteDB) GetScenarioStats(scenarioID string) (*models.ScenarioStats, error) {
	stats := &models.ScenarioStats{
		ScenarioID:  scenarioID,
		ByDataType:  make(map[string]int),
		ByExtension: make(map[string]int),
	}

	// Get counts and sizes
	query := `
		SELECT 
			COUNT(*),
			COUNT(CASE WHEN encryption_status = 'encrypted' THEN 1 END),
			COUNT(CASE WHEN encryption_status = 'pending' THEN 1 END),
			COUNT(CASE WHEN encryption_status = 'failed' THEN 1 END),
			SUM(size),
			SUM(CASE WHEN encryption_status = 'encrypted' THEN size ELSE 0 END)
		FROM files WHERE scenario_id = ?
	`
	err := s.db.QueryRow(query, scenarioID).Scan(
		&stats.TotalFiles,
		&stats.EncryptedFiles,
		&stats.PendingFiles,
		&stats.FailedFiles,
		&stats.TotalSize,
		&stats.EncryptedSize,
	)
	if err != nil {
		return nil, err
	}

	if stats.TotalFiles > 0 {
		stats.EncryptedPercent = float64(stats.EncryptedFiles) / float64(stats.TotalFiles) * 100
	}

	// Get by data type
	rows, err := s.db.Query(`
		SELECT data_type, COUNT(*) 
		FROM files WHERE scenario_id = ? 
		GROUP BY data_type
	`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dataType string
		var count int
		if err := rows.Scan(&dataType, &count); err == nil {
			stats.ByDataType[dataType] = count
		}
	}

	// Get by extension
	rows, err = s.db.Query(`
		SELECT extension, COUNT(*) 
		FROM files WHERE scenario_id = ? 
		GROUP BY extension
	`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var extension string
		var count int
		if err := rows.Scan(&extension, &count); err == nil {
			stats.ByExtension[extension] = count
		}
	}

	return stats, nil
}
