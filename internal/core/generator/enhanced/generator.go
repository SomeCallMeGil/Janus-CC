// Package enhanced provides the orchestrator for enhanced file generation
package enhanced

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"janus/internal/core/generator/filler"
	"janus/internal/core/generator/models"
	"janus/internal/core/generator/pii"
	"janus/internal/core/generator/resolver"
	"janus/internal/core/generator/validator"
	"janus/internal/core/tracker"
	dbmodels "janus/internal/database/models"
)

// Generator orchestrates enhanced file generation
type Generator struct {
	db           dbmodels.Database
	validator    *validator.Validator
	tracker      *tracker.Tracker
	seed         int64
}

// New creates a new enhanced generator
func New(db dbmodels.Database, seed int64) *Generator {
	return &Generator{
		db:        db,
		validator: validator.New(),
		tracker:   tracker.New(db),
		seed:      seed,
	}
}

// Progress represents generation progress
type Progress struct {
	Current       int     // Files completed
	Total         int     // Total files to generate
	Percent       float64 // Completion percentage
	CurrentFile   string  // Current file being generated
	BytesWritten  int64   // Total bytes written so far
	ElapsedTime   time.Duration
	EstimatedTime time.Duration
	Status        string  // "validating", "planning", "generating", "complete", "error"
}

// ProgressCallback is called during generation to report progress
type ProgressCallback func(Progress)

// GenerationResult contains the results of generation
type GenerationResult struct {
	Success      bool
	FilesCreated int
	BytesWritten int64
	Duration     time.Duration
	ScenarioID   string
	OutputPath   string
	Errors       []error
}

// Generate performs enhanced file generation with full validation
func (g *Generator) Generate(opts models.EnhancedGenerateOptions, callback ProgressCallback) (*GenerationResult, error) {
	startTime := time.Now()
	
	result := &GenerationResult{
		OutputPath: opts.OutputPath,
	}
	
	// Report validation phase
	if callback != nil {
		callback(Progress{Status: "validating"})
	}
	
	// Step 1: Validate all inputs
	validation := g.validator.ValidateAll(opts)
	if validation.HasErrors() {
		return nil, fmt.Errorf("validation failed:\n%s", validation.ErrorMessages())
	}
	
	// Step 2: Validate disk space
	diskValidation, diskInfo := g.validator.ValidateDiskSpace(opts)
	if diskValidation.HasErrors() {
		return nil, fmt.Errorf("disk space validation failed:\n%s", diskValidation.ErrorMessages())
	}
	
	// Log warnings if any
	if len(validation.Warnings) > 0 || len(diskValidation.Warnings) > 0 {
		fmt.Println("⚠️ Warnings:")
		if len(validation.Warnings) > 0 {
			fmt.Println(validation.WarningMessages())
		}
		if len(diskValidation.Warnings) > 0 {
			fmt.Println(diskValidation.WarningMessages())
		}
		fmt.Println()
	}
	
	// Report planning phase
	if callback != nil {
		callback(Progress{Status: "planning"})
	}
	
	// Step 3: Resolve constraints into concrete plan
	res := resolver.New(opts.Seed)
	plan, err := res.CreateGenerationPlan(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create generation plan: %w", err)
	}
	
	// Print plan summary
	fmt.Println("📋 Generation Plan:")
	fmt.Println(plan.Summary())
	fmt.Println()
	
	if diskInfo != nil {
		fmt.Printf("💾 Disk Space:\n")
		fmt.Printf("  Available: %s (after %s safety margin)\n", 
			models.FormatBytes(diskInfo.Available),
			models.FormatBytes(diskInfo.SafetyMargin))
		fmt.Printf("  Will use: %s (%.1f%% of available)\n",
			models.FormatBytes(diskInfo.RequiredSpace),
			float64(diskInfo.RequiredSpace)/float64(diskInfo.Available)*100)
		fmt.Printf("  Remaining: %s\n", 
			models.FormatBytes(diskInfo.Available-diskInfo.RequiredSpace))
		fmt.Println()
	}
	
	// Step 4: Create output directory
	if err := os.MkdirAll(opts.OutputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Report generating phase
	if callback != nil {
		callback(Progress{
			Total:  plan.Plan.FileCount,
			Status: "generating",
		})
	}
	
	// Step 5: Generate files
	monitor := &GenerationMonitor{
		StartTime:         startTime,
		DiskCheckInterval: 10 * time.Second,
		SafetyMargin:      diskInfo.SafetyMargin,
		OutputPath:        opts.OutputPath,
	}
	
	err = g.generateFiles(opts, plan, monitor, callback, result)
	if err != nil {
		return result, fmt.Errorf("generation failed: %w", err)
	}
	
	result.Success = true
	result.Duration = time.Since(startTime)
	
	// Report complete
	if callback != nil {
		callback(Progress{
			Current:      result.FilesCreated,
			Total:        plan.Plan.FileCount,
			Percent:      100.0,
			BytesWritten: result.BytesWritten,
			ElapsedTime:  result.Duration,
			Status:       "complete",
		})
	}
	
	return result, nil
}

// generateFiles generates all files according to plan
func (g *Generator) generateFiles(
	opts models.EnhancedGenerateOptions,
	plan *resolver.GenerationPlan,
	monitor *GenerationMonitor,
	callback ProgressCallback,
	result *GenerationResult,
) error {
	// Create generators
	fillerGen := filler.New(g.seed)
	piiGen := pii.New(true)
	
	// Generate each file
	for i := 0; i < plan.Plan.FileCount; i++ {
		// Check disk space periodically
		if err := monitor.CheckDiskSpace(); err != nil {
			return fmt.Errorf("disk space emergency: %w", err)
		}
		
		// Get file specs
		fileType := plan.FileTypes[i]
		fileSize := plan.FileSizes[i]
		format := plan.Formats[i%len(plan.Formats)] // Cycle through formats
		
		// Generate file path
		filePath := g.generateFilePath(opts.OutputPath, fileType, format, i, plan.Plan.DirectoryDepth)
		
		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("create directory: %w", err))
			continue
		}
		
		// Generate file based on type
		var err error
		switch fileType {
		case resolver.FileTypePII:
			err = g.generatePIIFile(filePath, format, fileSize, plan.PIIType, piiGen)
		case resolver.FileTypeFiller:
			err = fillerGen.GenerateToSize(filePath, format, fileSize)
		}
		
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("generate %s: %w", filePath, err))
			continue
		}
		
		// Track file size
		info, _ := os.Stat(filePath)
		if info != nil {
			result.BytesWritten += info.Size()
		}

		// Register file in DB if a scenario ID was provided
		if opts.ScenarioID != "" {
			dataType := opts.Distribution.PIIType
			if fileType == resolver.FileTypeFiller {
				dataType = "filler"
			}
			if trackErr := g.tracker.TrackFile(opts.ScenarioID, filePath, dataType); trackErr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("track %s: %w", filePath, trackErr))
			}
		}

		result.FilesCreated++
		
		// Report progress
		if callback != nil {
			elapsed := time.Since(monitor.StartTime)
			percentComplete := float64(i+1) / float64(plan.Plan.FileCount) * 100
			
			var estimatedTotal time.Duration
			if percentComplete > 0 {
				estimatedTotal = time.Duration(float64(elapsed) / percentComplete * 100)
			}
			
			callback(Progress{
				Current:       i + 1,
				Total:         plan.Plan.FileCount,
				Percent:       percentComplete,
				CurrentFile:   filepath.Base(filePath),
				BytesWritten:  result.BytesWritten,
				ElapsedTime:   elapsed,
				EstimatedTime: estimatedTotal,
				Status:        "generating",
			})
		}
	}
	
	return nil
}

// generateFilePath creates a file path with proper directory structure
func (g *Generator) generateFilePath(root string, fileType resolver.FileType, format string, index int, depth int) string {
	// Create subdirectory path
	subPath := root
	if depth > 1 {
		// Create random nested directories
		for d := 0; d < depth-1; d++ {
			dirName := fmt.Sprintf("dir_%d", (index+d)%100)
			subPath = filepath.Join(subPath, dirName)
		}
	}
	
	// Generate filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%d_%04d.%s", fileType.String(), timestamp, index, format)
	
	return filepath.Join(subPath, filename)
}

// generatePIIFile generates a PII file with target size
func (g *Generator) generatePIIFile(path, format string, targetSize int64, piiType string, piiGen *pii.Generator) error {
	// Estimate records needed (rough)
	bytesPerRecord := int64(500) // rough estimate
	recordCount := int(targetSize / bytesPerRecord)
	if recordCount < 1 {
		recordCount = 1
	}
	
	// Generate based on type and write directly
	switch piiType {
	case models.PIITypeStandard:
		records := piiGen.GenerateRecords(recordCount)
		switch format {
		case "csv":
			return g.writePIICSV(path, records)
		case "json":
			return g.writePIIJSON(path, records)
		case "txt":
			return g.writePIIText(path, records)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
		
	case models.PIITypeHealthcare:
		records := make([]*pii.MedicalRecord, recordCount)
		for i := 0; i < recordCount; i++ {
			records[i] = piiGen.GenerateMedicalRecord()
		}
		switch format {
		case "csv":
			return g.writeMedicalCSV(path, records)
		case "json":
			return g.writeMedicalJSON(path, records)
		case "txt":
			return g.writeMedicalText(path, records)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
		
	case models.PIITypeFinancial:
		records := make([]*pii.FinancialRecord, recordCount)
		for i := 0; i < recordCount; i++ {
			records[i] = piiGen.GenerateFinancialRecord()
		}
		switch format {
		case "csv":
			return g.writeFinancialCSV(path, records)
		case "json":
			return g.writeFinancialJSON(path, records)
		case "txt":
			return g.writeFinancialText(path, records)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
		
	default:
		// Fallback to standard
		records := piiGen.GenerateRecords(recordCount)
		switch format {
		case "csv":
			return g.writePIICSV(path, records)
		case "json":
			return g.writePIIJSON(path, records)
		case "txt":
			return g.writePIIText(path, records)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
	}
}

// writePIICSV writes PII records to CSV
func (g *Generator) writePIICSV(path string, records []*pii.Record) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	w := csv.NewWriter(f)
	defer w.Flush()
	
	// Write header
	header := []string{
		"FirstName", "LastName", "SSN", "DateOfBirth", "Email", "Phone",
		"Address", "City", "State", "ZipCode", "CreditCard", "DriversLicense",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	
	// Write records
	for _, record := range records {
		row := []string{
			record.FirstName,
			record.LastName,
			record.SSN,
			record.DateOfBirth,
			record.Email,
			record.Phone,
			record.Address,
			record.City,
			record.State,
			record.ZipCode,
			record.CreditCard,
			record.DriversLicense,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	
	return nil
}

// writePIIJSON writes PII records to JSON
func (g *Generator) writePIIJSON(path string, records []*pii.Record) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(records)
}

// writePIIText writes PII records to text
func (g *Generator) writePIIText(path string, records []*pii.Record) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	for _, record := range records {
		fmt.Fprintf(f, "Name: %s %s\n", record.FirstName, record.LastName)
		fmt.Fprintf(f, "SSN: %s\n", record.SSN)
		fmt.Fprintf(f, "DOB: %s\n", record.DateOfBirth)
		fmt.Fprintf(f, "Email: %s\n", record.Email)
		fmt.Fprintf(f, "Phone: %s\n", record.Phone)
		fmt.Fprintf(f, "Address: %s, %s, %s %s\n", 
			record.Address, record.City, record.State, record.ZipCode)
		fmt.Fprintf(f, "Credit Card: %s\n", record.CreditCard)
		fmt.Fprintf(f, "Driver's License: %s\n", record.DriversLicense)
		fmt.Fprintf(f, "\n---\n\n")
	}
	
	return nil
}

// writeMedicalCSV writes medical records to CSV
func (g *Generator) writeMedicalCSV(path string, records []*pii.MedicalRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	w := csv.NewWriter(f)
	defer w.Flush()
	
	// Write header
	header := []string{
		"FirstName", "LastName", "SSN", "DateOfBirth", "Email", "Phone",
		"Address", "City", "State", "ZipCode", "MRN", "InsuranceID",
		"InsuranceProvider", "PrimaryCarePhysician", "BloodType", "Allergies", "Medications",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	
	// Write records
	for _, record := range records {
		row := []string{
			record.FirstName,
			record.LastName,
			record.SSN,
			record.DateOfBirth,
			record.Email,
			record.Phone,
			record.Address,
			record.City,
			record.State,
			record.ZipCode,
			record.MRN,
			record.InsuranceID,
			record.InsuranceProvider,
			record.PrimaryCarePhysician,
			record.BloodType,
			record.Allergies,
			record.Medications,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	
	return nil
}

// writeMedicalJSON writes medical records to JSON
func (g *Generator) writeMedicalJSON(path string, records []*pii.MedicalRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(records)
}

// writeMedicalText writes medical records to text
func (g *Generator) writeMedicalText(path string, records []*pii.MedicalRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	for _, record := range records {
		fmt.Fprintf(f, "Name: %s %s\n", record.FirstName, record.LastName)
		fmt.Fprintf(f, "SSN: %s\n", record.SSN)
		fmt.Fprintf(f, "DOB: %s\n", record.DateOfBirth)
		fmt.Fprintf(f, "Email: %s\n", record.Email)
		fmt.Fprintf(f, "Phone: %s\n", record.Phone)
		fmt.Fprintf(f, "Address: %s, %s, %s %s\n", 
			record.Address, record.City, record.State, record.ZipCode)
		fmt.Fprintf(f, "MRN: %s\n", record.MRN)
		fmt.Fprintf(f, "Insurance: %s (%s)\n", record.InsuranceProvider, record.InsuranceID)
		fmt.Fprintf(f, "Primary Care: %s\n", record.PrimaryCarePhysician)
		fmt.Fprintf(f, "Blood Type: %s\n", record.BloodType)
		fmt.Fprintf(f, "Allergies: %s\n", record.Allergies)
		fmt.Fprintf(f, "Medications: %s\n", record.Medications)
		fmt.Fprintf(f, "\n---\n\n")
	}
	
	return nil
}

// writeFinancialCSV writes financial records to CSV
func (g *Generator) writeFinancialCSV(path string, records []*pii.FinancialRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	w := csv.NewWriter(f)
	defer w.Flush()
	
	// Write header
	header := []string{
		"FirstName", "LastName", "SSN", "DateOfBirth", "Email", "Phone",
		"Address", "City", "State", "ZipCode", "AccountNumber", "RoutingNumber",
		"BankName", "AccountType", "Balance", "CreditScore", "AnnualIncome",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	
	// Write records
	for _, record := range records {
		row := []string{
			record.FirstName,
			record.LastName,
			record.SSN,
			record.DateOfBirth,
			record.Email,
			record.Phone,
			record.Address,
			record.City,
			record.State,
			record.ZipCode,
			record.AccountNumber,
			record.RoutingNumber,
			record.BankName,
			record.AccountType,
			fmt.Sprintf("%.2f", record.Balance),
			fmt.Sprintf("%d", record.CreditScore),
			fmt.Sprintf("%d", record.AnnualIncome),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	
	return nil
}

// writeFinancialJSON writes financial records to JSON
func (g *Generator) writeFinancialJSON(path string, records []*pii.FinancialRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(records)
}

// writeFinancialText writes financial records to text
func (g *Generator) writeFinancialText(path string, records []*pii.FinancialRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	
	for _, record := range records {
		fmt.Fprintf(f, "Name: %s %s\n", record.FirstName, record.LastName)
		fmt.Fprintf(f, "SSN: %s\n", record.SSN)
		fmt.Fprintf(f, "DOB: %s\n", record.DateOfBirth)
		fmt.Fprintf(f, "Email: %s\n", record.Email)
		fmt.Fprintf(f, "Phone: %s\n", record.Phone)
		fmt.Fprintf(f, "Address: %s, %s, %s %s\n", 
			record.Address, record.City, record.State, record.ZipCode)
		fmt.Fprintf(f, "Bank: %s\n", record.BankName)
		fmt.Fprintf(f, "Account: %s (%s)\n", record.AccountNumber, record.AccountType)
		fmt.Fprintf(f, "Routing: %s\n", record.RoutingNumber)
		fmt.Fprintf(f, "Balance: $%.2f\n", record.Balance)
		fmt.Fprintf(f, "Credit Score: %d\n", record.CreditScore)
		fmt.Fprintf(f, "Annual Income: $%d\n", record.AnnualIncome)
		fmt.Fprintf(f, "\n---\n\n")
	}
	
	return nil
}

// GenerationMonitor monitors generation for issues
type GenerationMonitor struct {
	StartTime         time.Time
	LastDiskCheck     time.Time
	DiskCheckInterval time.Duration
	SafetyMargin      int64
	OutputPath        string
	mu                sync.Mutex
}

// CheckDiskSpace is implemented in platform-specific files:
// - monitor_unix.go for Linux/Mac
// - monitor_windows.go for Windows
