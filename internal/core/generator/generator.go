// Package generator orchestrates data generation across different data types.
package generator

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"janus/internal/core/generator/pii"
	"janus/internal/core/scenario"
	"janus/internal/core/tracker"
	"janus/internal/database/models"
)

// Generator orchestrates file generation
type Generator struct {
	db      models.Database
	tracker *tracker.Tracker
}

// New creates a new generator
func New(db models.Database) *Generator {
	return &Generator{
		db:      db,
		tracker: tracker.New(db),
	}
}

// GenerateOptions contains generation options
type GenerateOptions struct {
	ScenarioID string
	Config     *scenario.ScenarioConfig
	Quiet      bool
	Progress   ProgressCallback
}

// ProgressCallback reports generation progress
type ProgressCallback func(current, total int, message string)

// Generate generates files based on scenario configuration
func (g *Generator) Generate(opts GenerateOptions) error {
	cfg := opts.Config.Generation

	// Create root directory
	if err := os.MkdirAll(cfg.Root, 0755); err != nil {
		return fmt.Errorf("create root directory: %w", err)
	}

	// Generate files for each distribution
	totalFiles := 0
	for _, dist := range cfg.Distributions {
		for _, format := range dist.Formats {
			count := format.Count
			totalFiles += count

			if !opts.Quiet {
				fmt.Printf("Generating %d %s files (%s)...\n", count, dist.DataType, format.Format)
			}

			for i := 0; i < count; i++ {
				if opts.Progress != nil {
					opts.Progress(i+1, count, fmt.Sprintf("Generating %s file %d/%d", dist.DataType, i+1, count))
				}

				filePath, err := g.generateFile(cfg.Root, dist.DataType, format.Format, format.Templates, cfg.DirectoryDepth)
				if err != nil {
					return fmt.Errorf("generate file: %w", err)
				}

				// Track the file
				if err := g.tracker.TrackFile(opts.ScenarioID, filePath, dist.DataType); err != nil {
					return fmt.Errorf("track file: %w", err)
				}
			}
		}
	}

	if !opts.Quiet {
		fmt.Printf("Generated %d files in %s\n", totalFiles, cfg.Root)
	}

	return nil
}

// generateFile generates a single file
func (g *Generator) generateFile(root, dataType, format string, templates []string, depth int) (string, error) {
	// Create subdirectory structure
	subDir := g.generateSubdirectory(root, depth)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return "", fmt.Errorf("create subdirectory: %w", err)
	}

	// Generate filename
	filename := g.generateFilename(dataType, format)
	filePath := filepath.Join(subDir, filename)

	// Generate content based on data type and format
	switch dataType {
	case "pii":
		return filePath, g.generatePIIFile(filePath, format)
	case "healthcare":
		return filePath, g.generateHealthcareFile(filePath, format, templates)
	case "financial":
		return filePath, g.generateFinancialFile(filePath, format, templates)
	default:
		return filePath, g.generateGenericFile(filePath, format)
	}
}

// generateSubdirectory creates a random subdirectory path
func (g *Generator) generateSubdirectory(root string, depth int) string {
	if depth <= 1 {
		return root
	}

	path := root
	actualDepth := rand.Intn(depth) + 1
	
	for i := 0; i < actualDepth; i++ {
		dirName := fmt.Sprintf("dir_%d", rand.Intn(100))
		path = filepath.Join(path, dirName)
	}

	return path
}

// generateFilename generates a random filename
func (g *Generator) generateFilename(dataType, format string) string {
	timestamp := time.Now().Unix()
	random := rand.Intn(10000)
	
	return fmt.Sprintf("%s_%d_%04d.%s", dataType, timestamp, random, format)
}

// generatePIIFile generates a PII file
func (g *Generator) generatePIIFile(path, format string) error {
	piiGen := pii.New(true)

	switch format {
	case "csv":
		return g.generatePIICSV(path, piiGen)
	case "json":
		return g.generatePIIJSON(path, piiGen)
	case "txt":
		return g.generatePIIText(path, piiGen)
	default:
		return fmt.Errorf("unsupported PII format: %s", format)
	}
}

// generatePIICSV generates PII data in CSV format
func (g *Generator) generatePIICSV(path string, piiGen *pii.Generator) error {
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

	// Generate 10-50 records per file
	recordCount := rand.Intn(40) + 10
	for i := 0; i < recordCount; i++ {
		record := piiGen.GenerateRecord()
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

// generatePIIJSON generates PII data in JSON format
func (g *Generator) generatePIIJSON(path string, piiGen *pii.Generator) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	recordCount := rand.Intn(20) + 5
	records := piiGen.GenerateRecords(recordCount)

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

// generatePIIText generates PII data in text format
func (g *Generator) generatePIIText(path string, piiGen *pii.Generator) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	recordCount := rand.Intn(10) + 5
	records := piiGen.GenerateRecords(recordCount)

	for _, record := range records {
		fmt.Fprintf(f, "Name: %s\n", record.FullName)
		fmt.Fprintf(f, "SSN: %s\n", record.SSN)
		fmt.Fprintf(f, "Date of Birth: %s\n", record.DateOfBirth)
		fmt.Fprintf(f, "Email: %s\n", record.Email)
		fmt.Fprintf(f, "Phone: %s\n", record.Phone)
		fmt.Fprintf(f, "Address: %s, %s, %s %s\n", record.Address, record.City, record.State, record.ZipCode)
		fmt.Fprintf(f, "Credit Card: %s\n", record.CreditCard)
		fmt.Fprintf(f, "Drivers License: %s\n", record.DriversLicense)
		fmt.Fprintf(f, "\n---\n\n")
	}

	return nil
}

// generateHealthcareFile generates a healthcare file
func (g *Generator) generateHealthcareFile(path, format string, templates []string) error {
	switch format {
	case "json":
		return g.generateHealthcareJSON(path, templates)
	case "csv":
		return g.generateHealthcareCSV(path)
	case "txt":
		return g.generateHealthcareText(path)
	default:
		return fmt.Errorf("unsupported healthcare format: %s", format)
	}
}

// generateHealthcareJSON generates FHIR-style JSON
func (g *Generator) generateHealthcareJSON(path string, templates []string) error {
	piiGen := pii.New(true)
	record := piiGen.GenerateMedicalRecord()

	// Create a simplified FHIR Patient resource
	patient := map[string]interface{}{
		"resourceType": "Patient",
		"id":           record.MRN,
		"identifier": []map[string]interface{}{
			{
				"system": "http://hospital.example.org/patients",
				"value":  record.MRN,
			},
		},
		"name": []map[string]interface{}{
			{
				"use":    "official",
				"family": record.LastName,
				"given":  []string{record.FirstName},
			},
		},
		"telecom": []map[string]interface{}{
			{
				"system": "phone",
				"value":  record.Phone,
			},
			{
				"system": "email",
				"value":  record.Email,
			},
		},
		"birthDate": record.DateOfBirth,
		"address": []map[string]interface{}{
			{
				"line":       []string{record.Address},
				"city":       record.City,
				"state":      record.State,
				"postalCode": record.ZipCode,
			},
		},
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(patient)
}

// generateHealthcareCSV generates healthcare data in CSV
func (g *Generator) generateHealthcareCSV(path string) error {
	piiGen := pii.New(true)
	
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	header := []string{
		"MRN", "FirstName", "LastName", "DOB", "InsuranceID", "InsuranceProvider",
		"PrimaryCarePhysician", "BloodType", "Allergies", "Medications",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// Generate records
	recordCount := rand.Intn(30) + 10
	for i := 0; i < recordCount; i++ {
		record := piiGen.GenerateMedicalRecord()
		row := []string{
			record.MRN,
			record.FirstName,
			record.LastName,
			record.DateOfBirth,
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

// generateHealthcareText generates healthcare text file
func (g *Generator) generateHealthcareText(path string) error {
	piiGen := pii.New(true)
	record := piiGen.GenerateMedicalRecord()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "MEDICAL RECORD\n")
	fmt.Fprintf(f, "==============\n\n")
	fmt.Fprintf(f, "Patient Information:\n")
	fmt.Fprintf(f, "  MRN: %s\n", record.MRN)
	fmt.Fprintf(f, "  Name: %s\n", record.FullName)
	fmt.Fprintf(f, "  Date of Birth: %s\n", record.DateOfBirth)
	fmt.Fprintf(f, "  SSN: %s\n", record.SSN)
	fmt.Fprintf(f, "\nContact Information:\n")
	fmt.Fprintf(f, "  Phone: %s\n", record.Phone)
	fmt.Fprintf(f, "  Email: %s\n", record.Email)
	fmt.Fprintf(f, "  Address: %s, %s, %s %s\n", record.Address, record.City, record.State, record.ZipCode)
	fmt.Fprintf(f, "\nInsurance Information:\n")
	fmt.Fprintf(f, "  Insurance ID: %s\n", record.InsuranceID)
	fmt.Fprintf(f, "  Provider: %s\n", record.InsuranceProvider)
	fmt.Fprintf(f, "\nMedical Information:\n")
	fmt.Fprintf(f, "  Primary Care Physician: %s\n", record.PrimaryCarePhysician)
	fmt.Fprintf(f, "  Blood Type: %s\n", record.BloodType)
	fmt.Fprintf(f, "  Known Allergies: %s\n", record.Allergies)
	fmt.Fprintf(f, "  Current Medications: %s\n", record.Medications)

	return nil
}

// generateFinancialFile generates a financial file
func (g *Generator) generateFinancialFile(path, format string, templates []string) error {
	switch format {
	case "csv":
		return g.generateFinancialCSV(path)
	case "json":
		return g.generateFinancialJSON(path)
	case "txt":
		return g.generateFinancialText(path)
	default:
		return fmt.Errorf("unsupported financial format: %s", format)
	}
}

// generateFinancialCSV generates financial data in CSV
func (g *Generator) generateFinancialCSV(path string) error {
	piiGen := pii.New(true)
	
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	header := []string{
		"AccountNumber", "RoutingNumber", "FirstName", "LastName", "SSN",
		"BankName", "AccountType", "Balance", "CreditScore", "AnnualIncome",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// Generate records
	recordCount := rand.Intn(20) + 5
	for i := 0; i < recordCount; i++ {
		record := piiGen.GenerateFinancialRecord()
		row := []string{
			record.AccountNumber,
			record.RoutingNumber,
			record.FirstName,
			record.LastName,
			record.SSN,
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

// generateFinancialJSON generates financial data in JSON
func (g *Generator) generateFinancialJSON(path string) error {
	piiGen := pii.New(true)
	
	recordCount := rand.Intn(10) + 3
	var records []*pii.FinancialRecord
	for i := 0; i < recordCount; i++ {
		records = append(records, piiGen.GenerateFinancialRecord())
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

// generateFinancialText generates financial text file
func (g *Generator) generateFinancialText(path string) error {
	piiGen := pii.New(true)
	record := piiGen.GenerateFinancialRecord()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "FINANCIAL STATEMENT\n")
	fmt.Fprintf(f, "===================\n\n")
	fmt.Fprintf(f, "Account Holder:\n")
	fmt.Fprintf(f, "  Name: %s\n", record.FullName)
	fmt.Fprintf(f, "  SSN: %s\n", record.SSN)
	fmt.Fprintf(f, "  Date of Birth: %s\n", record.DateOfBirth)
	fmt.Fprintf(f, "\nAccount Information:\n")
	fmt.Fprintf(f, "  Bank: %s\n", record.BankName)
	fmt.Fprintf(f, "  Account Number: ****%s\n", record.AccountNumber[len(record.AccountNumber)-4:])
	fmt.Fprintf(f, "  Routing Number: %s\n", record.RoutingNumber)
	fmt.Fprintf(f, "  Account Type: %s\n", record.AccountType)
	fmt.Fprintf(f, "  Current Balance: $%.2f\n", record.Balance)
	fmt.Fprintf(f, "\nCredit Information:\n")
	fmt.Fprintf(f, "  Credit Score: %d\n", record.CreditScore)
	fmt.Fprintf(f, "  Annual Income: $%d\n", record.AnnualIncome)
	fmt.Fprintf(f, "  Credit Card: %s\n", record.CreditCard)

	return nil
}

// generateGenericFile generates a generic file with random data
func (g *Generator) generateGenericFile(path, format string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Generate random data
	size := rand.Intn(10240) + 1024 // 1KB - 10KB
	data := make([]byte, size)
	rand.Read(data)

	_, err = f.Write(data)
	return err
}
