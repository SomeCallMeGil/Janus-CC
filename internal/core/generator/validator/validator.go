// Package validator provides comprehensive validation for generation options
package validator

import (
	"fmt"
	"os"
	"path/filepath"

	"janus/internal/core/generator/models"
)

const (
	// Disk space constraints
	MinimumSafetyMarginPercent = 25.0                  // Keep 25% free
	AbsoluteSafetyMargin       = 5 * 1024 * 1024 * 1024 // or 5GB, whichever is larger
	
	// File size constraints
	MinFileSize = 1                                     // 1 byte minimum
	MaxFileSize = 10 * 1024 * 1024 * 1024                // 10GB per file warning threshold
	
	// Reasonable limits
	MaxTotalSize  = 1 * 1024 * 1024 * 1024 * 1024       // 1TB warning
	MaxFileCount  = 1000000                              // 1 million files warning
)

// Validator handles all validation logic
type Validator struct{}

// New creates a new validator
func New() *Validator {
	return &Validator{}
}

// ValidateAll performs all validations
func (v *Validator) ValidateAll(opts models.EnhancedGenerateOptions) models.ValidationResult {
	result := models.ValidationResult{Valid: true}
	
	// Run all validations
	v.validateConstraints(opts, &result)
	v.validateDistribution(opts, &result)
	v.validateFormats(opts, &result)
	v.validateOutputPath(opts, &result)
	v.validateLogicalConsistency(opts, &result)
	
	return result
}

// validateConstraints validates the constraint settings
func (v *Validator) validateConstraints(opts models.EnhancedGenerateOptions, result *models.ValidationResult) {
	c := opts.Constraints
	
	// 1. Primary mode must be valid
	validModes := []string{models.ConstraintModeSize, models.ConstraintModeCount, models.ConstraintModeManual}
	if !contains(validModes, c.PrimaryMode) {
		result.AddError("primary_mode", 
			fmt.Sprintf("Invalid primary mode '%s'. Must be: size, count, or manual", c.PrimaryMode),
			"INVALID_MODE")
	}
	
	// 2. File size validation
	if c.FileSizeMin < MinFileSize {
		result.AddError("file_size_min",
			fmt.Sprintf("Minimum file size must be at least %d byte", MinFileSize),
			"MIN_TOO_SMALL")
	}
	
	if c.FileSizeMin > c.FileSizeMax {
		result.AddError("file_size",
			fmt.Sprintf("Minimum file size (%s) cannot be greater than maximum (%s)",
				models.FormatBytes(c.FileSizeMin),
				models.FormatBytes(c.FileSizeMax)),
			"MIN_GT_MAX")
	}
	
	if c.FileSizeMax > MaxFileSize {
		result.AddWarning(
			fmt.Sprintf("Maximum file size is very large (%s). This may cause performance issues.",
				models.FormatBytes(c.FileSizeMax)),
			"high")
	}
	
	// 3. Mode-specific validation
	switch c.PrimaryMode {
	case models.ConstraintModeSize:
		if c.TotalSize < 1024 {
			result.AddError("total_size",
				"Total size must be at least 1KB",
				"SIZE_TOO_SMALL")
		}
		
		if c.TotalSize > MaxTotalSize {
			result.AddWarning(
				fmt.Sprintf("Total size is very large (%s). This may take a long time.",
					models.FormatBytes(c.TotalSize)),
				"high")
		}
		
	case models.ConstraintModeCount:
		if c.FileCount < 1 {
			result.AddError("file_count",
				"File count must be at least 1",
				"COUNT_TOO_SMALL")
		}
		
		if c.FileCount > MaxFileCount {
			result.AddWarning(
				fmt.Sprintf("File count is very large (%d). This may take a long time.",
					c.FileCount),
				"high")
		}
		
	case models.ConstraintModeManual:
		if c.FileCount < 1 {
			result.AddError("file_count",
				"File count must be at least 1 in manual mode",
				"COUNT_TOO_SMALL")
		}
	}
	
	// 4. Directory depth
	if c.DirectoryDepth < 1 {
		result.AddError("directory_depth",
			"Directory depth must be at least 1",
			"DEPTH_TOO_SMALL")
	}
	
	if c.DirectoryDepth > 10 {
		result.AddWarning(
			fmt.Sprintf("Directory depth of %d is very deep. This may cause path length issues on some systems.",
				c.DirectoryDepth),
			"medium")
	}
}

// validateDistribution validates content distribution
func (v *Validator) validateDistribution(opts models.EnhancedGenerateOptions, result *models.ValidationResult) {
	d := opts.Distribution
	
	// 1. Percentages must be 0-100
	if d.PIIPercent < 0 || d.PIIPercent > 100 {
		result.AddError("pii_percent",
			fmt.Sprintf("PII percentage must be between 0 and 100 (got %.1f)", d.PIIPercent),
			"INVALID_PERCENTAGE")
	}
	
	if d.FillerPercent < 0 || d.FillerPercent > 100 {
		result.AddError("filler_percent",
			fmt.Sprintf("Filler percentage must be between 0 and 100 (got %.1f)", d.FillerPercent),
			"INVALID_PERCENTAGE")
	}
	
	// 2. Must total 100%
	total := d.PIIPercent + d.FillerPercent
	if total != 100.0 {
		result.AddError("distribution",
			fmt.Sprintf("Distribution percentages must total 100%% (currently %.1f%%)", total),
			"INVALID_DISTRIBUTION")
	}
	
	// 3. If PII is selected, type must be valid
	if d.PIIPercent > 0 {
		validTypes := []string{models.PIITypeStandard, models.PIITypeHealthcare, models.PIITypeFinancial}
		if !contains(validTypes, d.PIIType) {
			result.AddError("pii_type",
				fmt.Sprintf("Invalid PII type '%s'. Must be: standard, healthcare, or financial", d.PIIType),
				"INVALID_PII_TYPE")
		}
	}
	
	// 4. Warn if 100% filler (no sensitive data)
	if d.FillerPercent == 100 {
		result.AddWarning(
			"Generation will contain only filler data (no PII). This may not be useful for security testing.",
			"low")
	}
	
	// 5. Warn if 100% PII (unrealistic)
	if d.PIIPercent == 100 {
		result.AddWarning(
			"Generation will contain 100% PII data. Real environments typically have mixed content.",
			"low")
	}
}

// validateFormats validates output formats
func (v *Validator) validateFormats(opts models.EnhancedGenerateOptions, result *models.ValidationResult) {
	if len(opts.Formats) == 0 {
		result.AddError("formats",
			"At least one output format must be selected",
			"NO_FORMATS")
		return
	}
	
	validFormats := []string{"csv", "json", "txt"}
	for _, format := range opts.Formats {
		if !contains(validFormats, format) {
			result.AddError("formats",
				fmt.Sprintf("Invalid format '%s'. Supported formats: csv, json, txt", format),
				"INVALID_FORMAT")
		}
	}
}

// validateOutputPath validates the output path
func (v *Validator) validateOutputPath(opts models.EnhancedGenerateOptions, result *models.ValidationResult) {
	if opts.OutputPath == "" {
		result.AddError("output_path",
			"Output path is required",
			"NO_OUTPUT_PATH")
		return
	}
	
	// Check if path is absolute or relative
	if !filepath.IsAbs(opts.OutputPath) {
		// Convert to absolute
		absPath, err := filepath.Abs(opts.OutputPath)
		if err != nil {
			result.AddWarning(
				fmt.Sprintf("Could not resolve absolute path: %v", err),
				"low")
		} else {
			// Just a warning, not an error
			result.AddWarning(
				fmt.Sprintf("Using relative path. Absolute path: %s", absPath),
				"low")
		}
	}
	
	// Check if parent directory exists
	parentDir := filepath.Dir(opts.OutputPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		result.AddError("output_path",
			fmt.Sprintf("Parent directory does not exist: %s", parentDir),
			"PARENT_NOT_EXISTS")
	}
	
	// Check if output path already exists
	if info, err := os.Stat(opts.OutputPath); err == nil {
		if info.IsDir() {
			// Directory exists - check if it's empty
			entries, _ := os.ReadDir(opts.OutputPath)
			if len(entries) > 0 {
				result.AddWarning(
					fmt.Sprintf("Output directory '%s' already exists and is not empty. Files may be overwritten.", opts.OutputPath),
					"medium")
			}
		} else {
			result.AddError("output_path",
				fmt.Sprintf("Path '%s' exists but is not a directory", opts.OutputPath),
				"NOT_A_DIRECTORY")
		}
	}
}

// validateLogicalConsistency checks for logical impossibilities
func (v *Validator) validateLogicalConsistency(opts models.EnhancedGenerateOptions, result *models.ValidationResult) {
	c := opts.Constraints
	
	// If size mode, check if we can create even one file
	if c.PrimaryMode == models.ConstraintModeSize {
		if c.TotalSize < c.FileSizeMin {
			result.AddError("constraints",
				fmt.Sprintf("Total size (%s) is smaller than minimum file size (%s). Cannot generate any files.",
					models.FormatBytes(c.TotalSize),
					models.FormatBytes(c.FileSizeMin)),
				"IMPOSSIBLE_CONSTRAINT")
		}
		
		// Estimate file count and warn if unrealistic
		avgFileSize := (c.FileSizeMin + c.FileSizeMax) / 2
		estimatedCount := c.TotalSize / avgFileSize
		
		if estimatedCount > MaxFileCount {
			result.AddWarning(
				fmt.Sprintf("This will generate approximately %d files, which is very large.", estimatedCount),
				"high")
		}
	}
	
	// If count mode, check total size isn't absurd
	if c.PrimaryMode == models.ConstraintModeCount {
		minTotalSize := int64(c.FileCount) * c.FileSizeMin
		maxTotalSize := int64(c.FileCount) * c.FileSizeMax
		
		if minTotalSize > MaxTotalSize {
			result.AddWarning(
				fmt.Sprintf("Minimum possible total size will be %s, which is very large.",
					models.FormatBytes(minTotalSize)),
				"high")
		}
		
		if maxTotalSize > 10*MaxTotalSize {
			result.AddWarning(
				fmt.Sprintf("Maximum possible total size could be %s, which is extremely large.",
					models.FormatBytes(maxTotalSize)),
				"high")
		}
	}
}

// ValidateDiskSpace checks if there's enough disk space
func (v *Validator) ValidateDiskSpace(opts models.EnhancedGenerateOptions) (models.ValidationResult, *models.DiskSpaceInfo) {
	result := models.ValidationResult{Valid: true}
	
	// Calculate estimated size
	estimatedSize := v.calculateEstimatedSize(opts)
	
	// Get disk info
	diskInfo, err := v.checkDiskSpace(opts.OutputPath, estimatedSize)
	if err != nil {
		result.AddError("disk_space",
			fmt.Sprintf("Cannot check disk space: %v", err),
			"DISK_CHECK_FAILED")
		return result, nil
	}
	
	// Check if sufficient
	if !diskInfo.Sufficient {
		result.AddError("disk_space",
			fmt.Sprintf("Insufficient disk space.\n"+
				"  Required: %s\n"+
				"  Available: %s (after %s safety margin)\n"+
				"  \n"+
				"  Disk Usage:\n"+
				"    Total: %s\n"+
				"    Used: %s (%.1f%%)\n"+
				"    Free: %s",
				models.FormatBytes(diskInfo.RequiredSpace),
				models.FormatBytes(diskInfo.Available),
				models.FormatBytes(diskInfo.SafetyMargin),
				models.FormatBytes(diskInfo.TotalSpace),
				models.FormatBytes(diskInfo.UsedSpace),
				diskInfo.UsagePercent,
				models.FormatBytes(diskInfo.FreeSpace)),
			"INSUFFICIENT_DISK_SPACE")
		return result, diskInfo
	}
	
	// Warnings for tight space
	if diskInfo.Available < diskInfo.RequiredSpace*2 {
		result.AddWarning(
			fmt.Sprintf("Disk space is tight. Available: %s, Required: %s",
				models.FormatBytes(diskInfo.Available),
				models.FormatBytes(diskInfo.RequiredSpace)),
			"medium")
	}
	
	// Warning for high disk usage
	if diskInfo.UsagePercent > 80 {
		result.AddWarning(
			fmt.Sprintf("Disk is %.1f%% full. After generation: %.1f%%",
				diskInfo.UsagePercent,
				diskInfo.AfterGenPercent),
			"high")
	}
	
	return result, diskInfo
}

// calculateEstimatedSize estimates the total size that will be generated
func (v *Validator) calculateEstimatedSize(opts models.EnhancedGenerateOptions) int64 {
	c := opts.Constraints
	
	switch c.PrimaryMode {
	case models.ConstraintModeSize:
		// Add 10% overhead for filesystem metadata
		return int64(float64(c.TotalSize) * 1.1)
		
	case models.ConstraintModeCount:
		// Average file size * count * 1.1 overhead
		avgSize := (c.FileSizeMin + c.FileSizeMax) / 2
		return int64(float64(c.FileCount) * float64(avgSize) * 1.1)
		
	case models.ConstraintModeManual:
		// Use minimum size * count * 1.1
		return int64(float64(c.FileCount) * float64(c.FileSizeMin) * 1.1)
	}
	
	return 0
}

// checkDiskSpace is implemented in platform-specific files:
// - validator_unix.go for Linux/Mac
// - validator_windows.go for Windows

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
