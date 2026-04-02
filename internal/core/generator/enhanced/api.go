// Package api provides a simple interface to enhanced generation
package enhanced

import (
	"context"
	"fmt"

	"janus/internal/core/generator/models"
	dbmodels "janus/internal/database/models"
)

// QuickGenerateOptions provides a simplified interface
type QuickGenerateOptions struct {
	// Scenario (set by server before generation; not required from client)
	ScenarioID string `json:"scenario_id,omitempty"`

	// Output
	Name       string `json:"name"`
	OutputPath string `json:"output_path"`

	// Primary constraint (pick ONE)
	TotalSize string `json:"total_size"` // e.g., "5GB", "100MB" (if set, this mode is used)
	FileCount int    `json:"file_count"` // e.g., 10000 (if TotalSize not set, this mode is used)

	// File size range
	FileSizeMin string `json:"file_size_min"` // e.g., "1KB"
	FileSizeMax string `json:"file_size_max"` // e.g., "10MB"

	// Content distribution (must total 100)
	PIIPercent    float64 `json:"pii_percent"`    // 0-100
	PIIType       string  `json:"pii_type"`       // "standard", "healthcare", "financial"
	FillerPercent float64 `json:"filler_percent"` // 0-100

	// Format
	Formats []string `json:"formats"` // ["csv", "json", "txt"]

	// Optional
	DirectoryDepth int   `json:"directory_depth"` // default: 3
	Seed           int64 `json:"seed"`            // for reproducible generation (0 = random)
	Workers        int   `json:"workers"`         // parallel file writers (0 = runtime.NumCPU)
}

// ToEnhancedOptions converts QuickGenerateOptions to EnhancedGenerateOptions
func (q *QuickGenerateOptions) ToEnhancedOptions() (models.EnhancedGenerateOptions, error) {
	opts := models.EnhancedGenerateOptions{
		ScenarioID:   q.ScenarioID,
		ScenarioName: q.Name,
		OutputPath:   q.OutputPath,
		Formats:      q.Formats,
		Seed:         q.Seed,
		Workers:      q.Workers,
		Distribution: models.ContentDistribution{
			PIIPercent:    q.PIIPercent,
			PIIType:       q.PIIType,
			FillerPercent: q.FillerPercent,
		},
	}
	
	// Parse file sizes
	var err error
	opts.Constraints.FileSizeMin, err = models.ParseSize(q.FileSizeMin)
	if err != nil {
		return opts, fmt.Errorf("invalid file size min: %w", err)
	}
	
	opts.Constraints.FileSizeMax, err = models.ParseSize(q.FileSizeMax)
	if err != nil {
		return opts, fmt.Errorf("invalid file size max: %w", err)
	}
	
	// Determine primary mode
	if q.TotalSize != "" {
		// Size mode
		opts.Constraints.PrimaryMode = models.ConstraintModeSize
		opts.Constraints.TotalSize, err = models.ParseSize(q.TotalSize)
		if err != nil {
			return opts, fmt.Errorf("invalid total size: %w", err)
		}
	} else if q.FileCount > 0 {
		// Count mode
		opts.Constraints.PrimaryMode = models.ConstraintModeCount
		opts.Constraints.FileCount = q.FileCount
	} else {
		return opts, fmt.Errorf("must specify either TotalSize or FileCount")
	}
	
	// Set directory depth (default 3)
	opts.Constraints.DirectoryDepth = q.DirectoryDepth
	if opts.Constraints.DirectoryDepth == 0 {
		opts.Constraints.DirectoryDepth = 3
	}
	
	// Default formats if not specified
	if len(opts.Formats) == 0 {
		opts.Formats = []string{"csv", "json", "txt"}
	}
	
	return opts, nil
}

// QuickGenerate provides a simplified generation interface.
// ctx and checkpoint control cancellation and pause respectively; pass nil for both
// if no job control is needed.
func QuickGenerate(ctx context.Context, db dbmodels.Database, opts QuickGenerateOptions, callback ProgressCallback, checkpoint func() error) (*GenerationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	enhancedOpts, err := opts.ToEnhancedOptions()
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	gen := New(db, opts.Seed)
	return gen.Generate(ctx, enhancedOpts, callback, checkpoint)
}

// Validate pre-validates options without generating
func Validate(db dbmodels.Database, opts QuickGenerateOptions) error {
	enhancedOpts, err := opts.ToEnhancedOptions()
	if err != nil {
		return err
	}
	
	gen := New(db, 0)
	
	// Validate inputs
	validation := gen.validator.ValidateAll(enhancedOpts)
	if validation.HasErrors() {
		return fmt.Errorf("validation failed:\n%s", validation.ErrorMessages())
	}
	
	// Validate disk space
	diskValidation, _ := gen.validator.ValidateDiskSpace(enhancedOpts)
	if diskValidation.HasErrors() {
		return fmt.Errorf("disk space validation failed:\n%s", diskValidation.ErrorMessages())
	}
	
	// Warnings are non-fatal; callers can inspect them via the ValidationResult if needed.
	// We don't print here — Validate is called from server context where stdout is not useful.
	return nil
}

// PrebuiltScenarios provides common scenarios
var PrebuiltScenarios = map[string]QuickGenerateOptions{
	"quick-pii-test": {
		Name:          "Quick PII Test",
		OutputPath:    "./payloads/quick-pii",
		FileCount:     1000,
		FileSizeMin:   "1KB",
		FileSizeMax:   "1MB",
		PIIPercent:    100,
		PIIType:       "standard",
		FillerPercent: 0,
		Formats:       []string{"csv", "json", "txt"},
	},
	
	"mixed-realistic": {
		Name:          "Mixed Realistic",
		OutputPath:    "./payloads/mixed",
		TotalSize:     "1GB",
		FileSizeMin:   "1KB",
		FileSizeMax:   "10MB",
		PIIPercent:    15,
		PIIType:       "standard",
		FillerPercent: 85,
		Formats:       []string{"csv", "json", "txt"},
	},
	
	"healthcare-large": {
		Name:          "Healthcare Large",
		OutputPath:    "./payloads/healthcare",
		TotalSize:     "5GB",
		FileSizeMin:   "10KB",
		FileSizeMax:   "50MB",
		PIIPercent:    30,
		PIIType:       "healthcare",
		FillerPercent: 70,
		Formats:       []string{"csv", "json"},
	},
	
	"financial-test": {
		Name:          "Financial Test",
		OutputPath:    "./payloads/financial",
		FileCount:     5000,
		FileSizeMin:   "5KB",
		FileSizeMax:   "5MB",
		PIIPercent:    40,
		PIIType:       "financial",
		FillerPercent: 60,
		Formats:       []string{"csv", "json", "txt"},
	},
}

// GetPrebuiltScenario returns a prebuilt scenario by name
func GetPrebuiltScenario(name string) (QuickGenerateOptions, bool) {
	opts, exists := PrebuiltScenarios[name]
	return opts, exists
}

// ListPrebuiltScenarios returns all prebuilt scenario names
func ListPrebuiltScenarios() []string {
	names := make([]string, 0, len(PrebuiltScenarios))
	for name := range PrebuiltScenarios {
		names = append(names, name)
	}
	return names
}
