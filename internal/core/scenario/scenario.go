// Package scenario provides scenario management functionality.
package scenario

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"janus/internal/database/models"
)

// Manager handles scenario operations
type Manager struct {
	db models.Database
}

// New creates a new scenario manager
func New(db models.Database) *Manager {
	return &Manager{db: db}
}

// ScenarioConfig represents the configuration for a scenario
type ScenarioConfig struct {
	Generation  GenerationConfig  `json:"generation"`
	Encryption  EncryptionConfig  `json:"encryption"`
	Tracking    TrackingConfig    `json:"tracking"`
	Cleanup     CleanupConfig     `json:"cleanup"`
}

// GenerationConfig configures data generation
type GenerationConfig struct {
	Root            string                 `json:"root"`
	TotalFiles      int                    `json:"total_files"`
	TotalSize       string                 `json:"total_size"`
	DirectoryDepth  int                    `json:"directory_depth"`
	Distributions   []DataDistribution     `json:"distributions"`
}

// DataDistribution defines data type distribution
type DataDistribution struct {
	DataType   string   `json:"data_type"` // pii, healthcare, financial
	Percentage float64  `json:"percentage"`
	Formats    []Format `json:"formats"`
}

// Format defines file format details
type Format struct {
	Format    string   `json:"format"` // pdf, csv, json, txt, xlsx
	Templates []string `json:"templates"`
	Count     int      `json:"count"`
}

// EncryptionConfig configures encryption
type EncryptionConfig struct {
	Mode            string             `json:"mode"` // full, partial
	PartialBytes    int64              `json:"partial_bytes"`
	PBKDFIterations int                `json:"pbkdf_iterations"`
	Schedule        []EncryptionSchedule `json:"schedule"`
}

// EncryptionSchedule defines when to encrypt
type EncryptionSchedule struct {
	Day        int      `json:"day"`
	Percentage float64  `json:"percentage"`
	Filters    EncryptionFilters `json:"filters"`
}

// EncryptionFilters for selective encryption
type EncryptionFilters struct {
	DataTypes  []string `json:"data_types"`
	Extensions []string `json:"extensions"`
}

// TrackingConfig configures file tracking
type TrackingConfig struct {
	HashAlgorithm string `json:"hash_algorithm"`
	ExportCSV     bool   `json:"export_csv"`
	CSVPath       string `json:"csv_path"`
}

// CleanupConfig configures cleanup behavior
type CleanupConfig struct {
	AutoDestroy   bool `json:"auto_destroy"`
	RetentionDays int  `json:"retention_days"`
}

// CreateRequest represents a scenario creation request
type CreateRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Template    string         `json:"template,omitempty"`
	Config      ScenarioConfig `json:"config"`
}

// Create creates a new scenario
func (m *Manager) Create(req CreateRequest) (*models.Scenario, error) {
	// Generate unique ID
	id := uuid.New().String()

	// Validate configuration
	if err := m.validateConfig(&req.Config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Serialize config
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	scenario := &models.Scenario{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Type:        "local", // or "distributed" based on config
		Config:      string(configJSON),
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.db.CreateScenario(scenario); err != nil {
		return nil, fmt.Errorf("create scenario: %w", err)
	}

	return scenario, nil
}

// Get retrieves a scenario by ID
func (m *Manager) Get(id string) (*models.Scenario, error) {
	return m.db.GetScenario(id)
}

// List returns all scenarios
func (m *Manager) List() ([]*models.Scenario, error) {
	return m.db.ListScenarios()
}

// Update updates a scenario
func (m *Manager) Update(id string, updates map[string]interface{}) (*models.Scenario, error) {
	scenario, err := m.db.GetScenario(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if name, ok := updates["name"].(string); ok {
		scenario.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		scenario.Description = desc
	}
	if status, ok := updates["status"].(string); ok {
		scenario.Status = status
	}

	scenario.UpdatedAt = time.Now()

	if err := m.db.UpdateScenario(scenario); err != nil {
		return nil, fmt.Errorf("update scenario: %w", err)
	}

	return scenario, nil
}

// Delete deletes a scenario
func (m *Manager) Delete(id string) error {
	return m.db.DeleteScenario(id)
}

// GetConfig retrieves and parses the scenario configuration
func (m *Manager) GetConfig(id string) (*ScenarioConfig, error) {
	scenario, err := m.db.GetScenario(id)
	if err != nil {
		return nil, err
	}

	var config ScenarioConfig
	if err := json.Unmarshal([]byte(scenario.Config), &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &config, nil
}

// UpdateStatus updates the scenario status
func (m *Manager) UpdateStatus(id string, status string) error {
	scenario, err := m.db.GetScenario(id)
	if err != nil {
		return err
	}

	scenario.Status = status
	scenario.UpdatedAt = time.Now()

	return m.db.UpdateScenario(scenario)
}

// GetStats returns statistics for a scenario
func (m *Manager) GetStats(id string) (*models.ScenarioStats, error) {
	return m.db.GetScenarioStats(id)
}

// validateConfig validates scenario configuration
func (m *Manager) validateConfig(cfg *ScenarioConfig) error {
	// Validate generation config
	if cfg.Generation.TotalFiles < 1 {
		return fmt.Errorf("total_files must be at least 1")
	}

	if cfg.Generation.DirectoryDepth < 1 || cfg.Generation.DirectoryDepth > 10 {
		return fmt.Errorf("directory_depth must be between 1 and 10")
	}

	// Validate distribution percentages sum to ~100
	var totalPercentage float64
	for _, dist := range cfg.Generation.Distributions {
		totalPercentage += dist.Percentage
		
		if dist.Percentage < 0 || dist.Percentage > 100 {
			return fmt.Errorf("distribution percentage must be 0-100, got %.2f", dist.Percentage)
		}
	}

	if totalPercentage > 100.01 || totalPercentage < 99.99 {
		return fmt.Errorf("distribution percentages must sum to 100, got %.2f", totalPercentage)
	}

	// Validate encryption schedule
	for _, sched := range cfg.Encryption.Schedule {
		if sched.Percentage < 0 || sched.Percentage > 100 {
			return fmt.Errorf("schedule percentage must be 0-100, got %.2f", sched.Percentage)
		}
	}

	return nil
}

// LoadTemplate loads a predefined scenario template
func (m *Manager) LoadTemplate(name string) (*ScenarioConfig, error) {
	templates := map[string]ScenarioConfig{
		"healthcare-basic": {
			Generation: GenerationConfig{
				Root:           "./payloads/healthcare-basic",
				TotalFiles:     100,
				TotalSize:      "10MB",
				DirectoryDepth: 2,
				Distributions: []DataDistribution{
					{
						DataType:   "healthcare",
						Percentage: 70,
						Formats: []Format{
							{Format: "pdf", Templates: []string{"patient_record"}, Count: 50},
							{Format: "json", Templates: []string{"fhir_patient"}, Count: 20},
						},
					},
					{
						DataType:   "pii",
						Percentage: 30,
						Formats: []Format{
							{Format: "csv", Count: 30},
						},
					},
				},
			},
			Encryption: EncryptionConfig{
				Mode:            "partial",
				PartialBytes:    4096,
				PBKDFIterations: 100000,
			},
		},
		"healthcare-large": {
			Generation: GenerationConfig{
				Root:           "./payloads/healthcare-large",
				TotalFiles:     10000,
				TotalSize:      "1GB",
				DirectoryDepth: 4,
				Distributions: []DataDistribution{
					{
						DataType:   "healthcare",
						Percentage: 60,
						Formats: []Format{
							{Format: "pdf", Templates: []string{"patient_record", "lab_result", "discharge"}, Count: 4000},
							{Format: "json", Templates: []string{"fhir_patient", "fhir_observation"}, Count: 2000},
						},
					},
					{
						DataType:   "financial",
						Percentage: 25,
						Formats: []Format{
							{Format: "pdf", Templates: []string{"invoice", "statement"}, Count: 2000},
							{Format: "xlsx", Count: 500},
						},
					},
					{
						DataType:   "pii",
						Percentage: 15,
						Formats: []Format{
							{Format: "csv", Count: 1000},
							{Format: "json", Count: 500},
						},
					},
				},
			},
			Encryption: EncryptionConfig{
				Mode:            "partial",
				PartialBytes:    4096,
				PBKDFIterations: 100000,
				Schedule: []EncryptionSchedule{
					{Day: 1, Percentage: 10},
					{Day: 3, Percentage: 20},
					{Day: 5, Percentage: 30},
					{Day: 7, Percentage: 40},
				},
			},
		},
		"financial-tax": {
			Generation: GenerationConfig{
				Root:           "./payloads/financial-tax",
				TotalFiles:     500,
				TotalSize:      "50MB",
				DirectoryDepth: 2,
				Distributions: []DataDistribution{
					{
						DataType:   "financial",
						Percentage: 100,
						Formats: []Format{
							{Format: "pdf", Templates: []string{"w2", "1099", "bank_statement"}, Count: 400},
							{Format: "xlsx", Templates: []string{"tax_summary"}, Count: 100},
						},
					},
				},
			},
			Encryption: EncryptionConfig{
				Mode:            "full",
				PBKDFIterations: 100000,
			},
		},
		"pii-only": {
			Generation: GenerationConfig{
				Root:           "./payloads/pii",
				TotalFiles:     1000,
				TotalSize:      "20MB",
				DirectoryDepth: 2,
				Distributions: []DataDistribution{
					{
						DataType:   "pii",
						Percentage: 100,
						Formats: []Format{
							{Format: "csv", Count: 500},
							{Format: "json", Count: 300},
							{Format: "txt", Count: 200},
						},
					},
				},
			},
			Encryption: EncryptionConfig{
				Mode:            "partial",
				PartialBytes:    4096,
				PBKDFIterations: 100000,
			},
		},
	}

	cfg, ok := templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	return &cfg, nil
}

// ListTemplates returns available template names
func (m *Manager) ListTemplates() []string {
	return []string{
		"healthcare-basic",
		"healthcare-large",
		"financial-tax",
		"pii-only",
	}
}

// CreateFromTemplate creates a scenario from a template
func (m *Manager) CreateFromTemplate(name, templateName string) (*models.Scenario, error) {
	cfg, err := m.LoadTemplate(templateName)
	if err != nil {
		return nil, err
	}

	req := CreateRequest{
		Name:        name,
		Description: fmt.Sprintf("Scenario created from template: %s", templateName),
		Template:    templateName,
		Config:      *cfg,
	}

	return m.Create(req)
}
