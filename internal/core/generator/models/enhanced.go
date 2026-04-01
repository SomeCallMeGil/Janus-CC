// Package models provides enhanced data structures for flexible generation
package models

import (
	"fmt"
	"strings"
)

// Primary constraint modes
const (
	ConstraintModeSize   = "size"   // User specifies total size, we calculate file count
	ConstraintModeCount  = "count"  // User specifies file count, we calculate sizes
	ConstraintModeManual = "manual" // User controls everything
)

// PII types
const (
	PIITypeStandard   = "standard"
	PIITypeHealthcare = "healthcare"
	PIITypeFinancial  = "financial"
)

// GenerateConstraints defines the primary constraints for generation
type GenerateConstraints struct {
	PrimaryMode   string `json:"primary_mode"`    // "size", "count", or "manual"
	TotalSize     int64  `json:"total_size"`      // bytes (used if mode = "size")
	FileCount     int    `json:"file_count"`      // number of files (used if mode = "count")
	FileSizeMin   int64  `json:"file_size_min"`   // minimum file size in bytes
	FileSizeMax   int64  `json:"file_size_max"`   // maximum file size in bytes
	DirectoryDepth int   `json:"directory_depth"` // how deep to nest folders
}

// ContentDistribution defines how content is distributed
type ContentDistribution struct {
	PIIPercent    float64 `json:"pii_percent"`    // 0-100
	PIIType       string  `json:"pii_type"`       // "standard", "healthcare", "financial"
	FillerPercent float64 `json:"filler_percent"` // 0-100
}

// EnhancedGenerateOptions combines all options for generation
type EnhancedGenerateOptions struct {
	ScenarioID   string              `json:"scenario_id"`   // DB scenario ID for file tracking (empty = no tracking)
	ScenarioName string              `json:"scenario_name"`
	OutputPath   string              `json:"output_path"`
	Constraints  GenerateConstraints `json:"constraints"`
	Distribution ContentDistribution `json:"distribution"`
	Formats      []string            `json:"formats"` // csv, json, txt
	Seed         int64               `json:"seed"`    // for reproducible generation (0 = random)
	Workers      int                 `json:"workers"` // parallel file writers (0 = runtime.NumCPU)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationWarning represents a non-blocking warning
type ValidationWarning struct {
	Message  string `json:"message"`
	Severity string `json:"severity"` // "low", "medium", "high"
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(field, message, code string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

// AddWarning adds a warning to the validation result
func (vr *ValidationResult) AddWarning(message, severity string) {
	vr.Warnings = append(vr.Warnings, ValidationWarning{
		Message:  message,
		Severity: severity,
	})
}

// HasErrors returns true if there are validation errors
func (vr *ValidationResult) HasErrors() bool {
	return !vr.Valid || len(vr.Errors) > 0
}

// ErrorMessages returns all error messages as a single string
func (vr *ValidationResult) ErrorMessages() string {
	if len(vr.Errors) == 0 {
		return ""
	}
	
	var messages []string
	for _, err := range vr.Errors {
		messages = append(messages, fmt.Sprintf("• %s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "\n")
}

// WarningMessages returns all warning messages as a single string
func (vr *ValidationResult) WarningMessages() string {
	if len(vr.Warnings) == 0 {
		return ""
	}
	
	var messages []string
	for _, warn := range vr.Warnings {
		icon := "⚠️"
		if warn.Severity == "high" {
			icon = "⚠️⚠️"
		}
		messages = append(messages, fmt.Sprintf("%s %s", icon, warn.Message))
	}
	return strings.Join(messages, "\n")
}

// ResolvedPlan represents the calculated generation plan
type ResolvedPlan struct {
	FileCount        int     `json:"file_count"`
	PIIFileCount     int     `json:"pii_file_count"`
	FillerFileCount  int     `json:"filler_file_count"`
	EstimatedSize    int64   `json:"estimated_size"`
	FileSizeMin      int64   `json:"file_size_min"`
	FileSizeMax      int64   `json:"file_size_max"`
	AvgFileSize      int64   `json:"avg_file_size"`
	DirectoryDepth   int     `json:"directory_depth"`
	OutputPath       string  `json:"output_path"`
}

// DiskSpaceInfo contains disk space information
type DiskSpaceInfo struct {
	TotalSpace     int64   `json:"total_space"`
	UsedSpace      int64   `json:"used_space"`
	FreeSpace      int64   `json:"free_space"`
	RequiredSpace  int64   `json:"required_space"`
	SafetyMargin   int64   `json:"safety_margin"`
	Available      int64   `json:"available"` // Free - Safety
	Sufficient     bool    `json:"sufficient"`
	UsagePercent   float64 `json:"usage_percent"`
	AfterGenPercent float64 `json:"after_gen_percent"`
}

// FormatBytes formats bytes into human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// ParseSize parses size strings like "5GB", "100MB", "1KB" to bytes
func ParseSize(sizeStr string) (int64, error) {
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}
	
	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(sizeStr, suffix) {
			numStr := strings.TrimSuffix(sizeStr, suffix)
			numStr = strings.TrimSpace(numStr)
			
			var num float64
			_, err := fmt.Sscanf(numStr, "%f", &num)
			if err != nil {
				return 0, fmt.Errorf("invalid size format: %s", sizeStr)
			}
			
			return int64(num * float64(multiplier)), nil
		}
	}
	
	return 0, fmt.Errorf("unrecognized size format: %s (use B, KB, MB, GB, TB)", sizeStr)
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
