// Package resolver calculates the actual generation plan from user constraints
package resolver

import (
	"fmt"
	"math/rand"

	"janus/internal/core/generator/models"
)

// Resolver resolves constraints into an actual generation plan
type Resolver struct {
	seed int64
	rng  *rand.Rand
}

// New creates a new resolver
func New(seed int64) *Resolver {
	var rng *rand.Rand
	if seed != 0 {
		rng = rand.New(rand.NewSource(seed))
	} else {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	
	return &Resolver{
		seed: seed,
		rng:  rng,
	}
}

// Resolve converts user options into a concrete generation plan
func (r *Resolver) Resolve(opts models.EnhancedGenerateOptions) (*models.ResolvedPlan, error) {
	plan := &models.ResolvedPlan{
		OutputPath:     opts.OutputPath,
		FileSizeMin:    opts.Constraints.FileSizeMin,
		FileSizeMax:    opts.Constraints.FileSizeMax,
		DirectoryDepth: opts.Constraints.DirectoryDepth,
	}
	
	// Calculate based on primary mode
	switch opts.Constraints.PrimaryMode {
	case models.ConstraintModeSize:
		r.resolveSizeConstrained(opts, plan)
	case models.ConstraintModeCount:
		r.resolveCountConstrained(opts, plan)
	case models.ConstraintModeManual:
		r.resolveManual(opts, plan)
	default:
		return nil, fmt.Errorf("invalid primary mode: %s", opts.Constraints.PrimaryMode)
	}
	
	// Split by distribution
	r.applyDistribution(opts.Distribution, plan)
	
	return plan, nil
}

// resolveSizeConstrained handles "total size" as primary constraint
func (r *Resolver) resolveSizeConstrained(opts models.EnhancedGenerateOptions, plan *models.ResolvedPlan) {
	c := opts.Constraints
	
	// Calculate average file size
	plan.AvgFileSize = (c.FileSizeMin + c.FileSizeMax) / 2
	
	// Estimate file count to meet total size
	// Use average size to estimate count
	plan.FileCount = int(c.TotalSize / plan.AvgFileSize)
	
	// Ensure at least 1 file
	if plan.FileCount < 1 {
		plan.FileCount = 1
	}
	
	// Set estimated size to target
	plan.EstimatedSize = c.TotalSize
}

// resolveCountConstrained handles "file count" as primary constraint
func (r *Resolver) resolveCountConstrained(opts models.EnhancedGenerateOptions, plan *models.ResolvedPlan) {
	c := opts.Constraints
	
	// File count is fixed
	plan.FileCount = c.FileCount
	
	// Calculate average file size
	plan.AvgFileSize = (c.FileSizeMin + c.FileSizeMax) / 2
	
	// Estimate total size (files will be random between min/max)
	plan.EstimatedSize = int64(plan.FileCount) * plan.AvgFileSize
}

// resolveManual handles manual mode where user sets everything
func (r *Resolver) resolveManual(opts models.EnhancedGenerateOptions, plan *models.ResolvedPlan) {
	c := opts.Constraints
	
	// User specifies both count and size constraints
	plan.FileCount = c.FileCount
	plan.AvgFileSize = (c.FileSizeMin + c.FileSizeMax) / 2
	plan.EstimatedSize = int64(plan.FileCount) * plan.AvgFileSize
}

// applyDistribution splits files by PII vs filler
func (r *Resolver) applyDistribution(dist models.ContentDistribution, plan *models.ResolvedPlan) {
	// Calculate PII file count
	plan.PIIFileCount = int(float64(plan.FileCount) * dist.PIIPercent / 100.0)
	
	// Rest are filler
	plan.FillerFileCount = plan.FileCount - plan.PIIFileCount
	
	// Ensure counts add up (handle rounding)
	if plan.PIIFileCount + plan.FillerFileCount != plan.FileCount {
		// Give the extra file to whichever has higher percentage
		if dist.PIIPercent > dist.FillerPercent {
			plan.PIIFileCount++
		} else {
			plan.FillerFileCount++
		}
	}
}

// GenerateFileSizes generates random file sizes for the plan
func (r *Resolver) GenerateFileSizes(plan *models.ResolvedPlan, mode string) []int64 {
	sizes := make([]int64, plan.FileCount)
	
	switch mode {
	case models.ConstraintModeSize:
		// For size-constrained, distribute the total size across files
		sizes = r.distributeSize(plan.EstimatedSize, plan.FileCount, plan.FileSizeMin, plan.FileSizeMax)
		
	case models.ConstraintModeCount, models.ConstraintModeManual:
		// For count-constrained, random sizes between min and max
		for i := 0; i < plan.FileCount; i++ {
			sizes[i] = r.randomSizeBetween(plan.FileSizeMin, plan.FileSizeMax)
		}
	}
	
	return sizes
}

// distributeSize distributes total size across N files, respecting min/max
func (r *Resolver) distributeSize(totalSize int64, fileCount int, minSize, maxSize int64) []int64 {
	sizes := make([]int64, fileCount)
	remaining := totalSize
	
	// First pass: give each file minimum size
	for i := 0; i < fileCount; i++ {
		sizes[i] = minSize
		remaining -= minSize
	}
	
	// Second pass: distribute remaining randomly
	for remaining > 0 && fileCount > 0 {
		// Pick random file
		idx := r.rng.Intn(fileCount)
		
		// How much can we add?
		canAdd := maxSize - sizes[idx]
		if canAdd <= 0 {
			continue
		}
		
		// Add some amount
		addAmount := r.rng.Int63n(canAdd) + 1
		if addAmount > remaining {
			addAmount = remaining
		}
		
		sizes[idx] += addAmount
		remaining -= addAmount
	}
	
	// If we still have remaining (all files at max), distribute across all files
	if remaining > 0 {
		perFile := remaining / int64(fileCount)
		for i := 0; i < fileCount; i++ {
			sizes[i] += perFile
		}
	}
	
	return sizes
}

// randomSizeBetween generates a random size between min and max
func (r *Resolver) randomSizeBetween(min, max int64) int64 {
	if min == max {
		return min
	}
	
	diff := max - min
	return min + r.rng.Int63n(diff)
}

// GenerateFileDistribution creates a plan for which files are PII vs filler
func (r *Resolver) GenerateFileDistribution(plan *models.ResolvedPlan) []FileType {
	distribution := make([]FileType, plan.FileCount)
	
	// Mark PII files
	for i := 0; i < plan.PIIFileCount; i++ {
		distribution[i] = FileTypePII
	}
	
	// Mark filler files
	for i := plan.PIIFileCount; i < plan.FileCount; i++ {
		distribution[i] = FileTypeFiller
	}
	
	// Shuffle to randomize distribution
	r.rng.Shuffle(len(distribution), func(i, j int) {
		distribution[i], distribution[j] = distribution[j], distribution[i]
	})
	
	return distribution
}

// FileType represents what type of content a file should have
type FileType int

const (
	FileTypePII FileType = iota
	FileTypeFiller
)

func (ft FileType) String() string {
	switch ft {
	case FileTypePII:
		return "pii"
	case FileTypeFiller:
		return "filler"
	default:
		return "unknown"
	}
}

// GenerationPlan combines resolved plan with file-level details
type GenerationPlan struct {
	Plan         *models.ResolvedPlan
	FileSizes    []int64
	FileTypes    []FileType
	Formats      []string
	PIIType      string
	Seed         int64
}

// CreateGenerationPlan creates a complete plan ready for execution
func (r *Resolver) CreateGenerationPlan(opts models.EnhancedGenerateOptions) (*GenerationPlan, error) {
	// Resolve constraints
	plan, err := r.Resolve(opts)
	if err != nil {
		return nil, err
	}
	
	// Generate file sizes
	sizes := r.GenerateFileSizes(plan, opts.Constraints.PrimaryMode)
	
	// Generate distribution
	types := r.GenerateFileDistribution(plan)
	
	return &GenerationPlan{
		Plan:      plan,
		FileSizes: sizes,
		FileTypes: types,
		Formats:   opts.Formats,
		PIIType:   opts.Distribution.PIIType,
		Seed:      r.seed,
	}, nil
}

// Summary returns a human-readable summary of the plan
func (gp *GenerationPlan) Summary() string {
	return fmt.Sprintf(`Generation Plan:
  Output: %s
  Total Files: %d
    • PII (%s): %d files
    • Filler: %d files
  Estimated Size: %s
  File Size Range: %s - %s
  Directory Depth: %d levels
  Formats: %v
  Seed: %d (reproducible: %v)`,
		gp.Plan.OutputPath,
		gp.Plan.FileCount,
		gp.PIIType,
		gp.Plan.PIIFileCount,
		gp.Plan.FillerFileCount,
		models.FormatBytes(gp.Plan.EstimatedSize),
		models.FormatBytes(gp.Plan.FileSizeMin),
		models.FormatBytes(gp.Plan.FileSizeMax),
		gp.Plan.DirectoryDepth,
		gp.Formats,
		gp.Seed,
		gp.Seed != 0,
	)
}
