// Package pii provides PII (Personally Identifiable Information) data generation.
package pii

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

func init() {
	gofakeit.Seed(time.Now().UnixNano())
}

// Record represents a PII data record
type Record struct {
	FirstName      string `json:"first_name" csv:"first_name"`
	LastName       string `json:"last_name" csv:"last_name"`
	FullName       string `json:"full_name" csv:"full_name"`
	SSN            string `json:"ssn" csv:"ssn"`
	DateOfBirth    string `json:"date_of_birth" csv:"date_of_birth"`
	Email          string `json:"email" csv:"email"`
	Phone          string `json:"phone" csv:"phone"`
	Address        string `json:"address" csv:"address"`
	City           string `json:"city" csv:"city"`
	State          string `json:"state" csv:"state"`
	ZipCode        string `json:"zip_code" csv:"zip_code"`
	CreditCard     string `json:"credit_card" csv:"credit_card"`
	DriversLicense string `json:"drivers_license" csv:"drivers_license"`
	PassportNumber string `json:"passport_number" csv:"passport_number"`
}

// Generator generates PII data
type Generator struct {
	realistic bool
}

// New creates a new PII generator
func New(realistic bool) *Generator {
	return &Generator{
		realistic: realistic,
	}
}

// GenerateRecord generates a single PII record
func (g *Generator) GenerateRecord() *Record {
	firstName := gofakeit.FirstName()
	lastName := gofakeit.LastName()

	return &Record{
		FirstName:      firstName,
		LastName:       lastName,
		FullName:       fmt.Sprintf("%s %s", firstName, lastName),
		SSN:            g.generateSSN(),
		DateOfBirth:    g.generateDOB(),
		Email:          g.generateEmail(firstName, lastName),
		Phone:          g.generatePhone(),
		Address:        gofakeit.Address().Address,
		City:           gofakeit.City(),
		State:          gofakeit.StateAbr(),
		ZipCode:        gofakeit.Zip(),
		CreditCard:     g.generateCreditCard(),
		DriversLicense: g.generateDriversLicense(),
		PassportNumber: g.generatePassport(),
	}
}

// GenerateRecords generates multiple PII records
func (g *Generator) GenerateRecords(count int) []*Record {
	records := make([]*Record, count)
	for i := 0; i < count; i++ {
		records[i] = g.GenerateRecord()
	}
	return records
}

// generateSSN generates a realistic SSN
func (g *Generator) generateSSN() string {
	// Format: XXX-XX-XXXX
	// Avoid known invalid ranges
	area := rand.Intn(899) + 100 // 100-998 (avoiding 666 and 900+)
	if area == 666 {
		area = 667
	}
	group := rand.Intn(99) + 1    // 01-99
	serial := rand.Intn(9999) + 1 // 0001-9999

	return fmt.Sprintf("%03d-%02d-%04d", area, group, serial)
}

// generateDOB generates a date of birth
func (g *Generator) generateDOB() string {
	// Generate age between 18-80
	years := rand.Intn(62) + 18
	dob := time.Now().AddDate(-years, -rand.Intn(12), -rand.Intn(28))
	return dob.Format("2006-01-02")
}

// generateEmail generates an email address
func (g *Generator) generateEmail(firstName, lastName string) string {
	domains := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "aol.com", "icloud.com"}
	
	// Various email patterns
	patterns := []string{
		"%s.%s@%s",
		"%s_%s@%s",
		"%s%s@%s",
		"%s.%s%d@%s",
	}
	
	pattern := patterns[rand.Intn(len(patterns))]
	domain := domains[rand.Intn(len(domains))]
	
	first := strings.ToLower(firstName)
	last := strings.ToLower(lastName)
	
	if strings.Contains(pattern, "%d") {
		return fmt.Sprintf(pattern, first, last, rand.Intn(999), domain)
	}
	return fmt.Sprintf(pattern, first, last, domain)
}

// generatePhone generates a US phone number
func (g *Generator) generatePhone() string {
	// Format: (XXX) XXX-XXXX
	area := rand.Intn(800) + 200 // 200-999
	exchange := rand.Intn(800) + 200
	line := rand.Intn(10000)
	
	return fmt.Sprintf("(%03d) %03d-%04d", area, exchange, line)
}

// generateCreditCard generates a valid-looking credit card number (Luhn algorithm)
func (g *Generator) generateCreditCard() string {
	// Generate Visa (starts with 4)
	length := 16
	
	// Generate random digits
	digits := make([]int, length-1)
	for i := 0; i < length-1; i++ {
		if i == 0 {
			digits[i] = 4 // Visa starts with 4
		} else {
			digits[i] = rand.Intn(10)
		}
	}
	
	// Calculate Luhn check digit
	checkDigit := g.calculateLuhn(digits)
	digits = append(digits, checkDigit)
	
	// Format as XXXX-XXXX-XXXX-XXXX
	var parts []string
	for i := 0; i < 4; i++ {
		part := ""
		for j := 0; j < 4; j++ {
			part += fmt.Sprintf("%d", digits[i*4+j])
		}
		parts = append(parts, part)
	}
	
	return strings.Join(parts, "-")
}

// calculateLuhn calculates Luhn check digit
func (g *Generator) calculateLuhn(digits []int) int {
	sum := 0
	isEven := true // Start from right, so first digit from left is "even" position
	
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		
		if isEven {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		
		sum += digit
		isEven = !isEven
	}
	
	checkDigit := (10 - (sum % 10)) % 10
	return checkDigit
}

// generateDriversLicense generates a drivers license number
func (g *Generator) generateDriversLicense() string {
	// Format varies by state, using generic format: L#######
	state := gofakeit.StateAbr()
	number := rand.Intn(9999999)
	
	return fmt.Sprintf("%s%07d", state, number)
}

// generatePassport generates a passport number
func (g *Generator) generatePassport() string {
	// US passport format: C########
	letter := string(rune('A' + rand.Intn(26)))
	number := rand.Intn(99999999)
	
	return fmt.Sprintf("%s%08d", letter, number)
}

// GenerateEmployeeRecord generates employment-related PII
type EmployeeRecord struct {
	Record
	EmployeeID     string `json:"employee_id" csv:"employee_id"`
	Department     string `json:"department" csv:"department"`
	JobTitle       string `json:"job_title" csv:"job_title"`
	Salary         int    `json:"salary" csv:"salary"`
	HireDate       string `json:"hire_date" csv:"hire_date"`
	ManagerName    string `json:"manager_name" csv:"manager_name"`
	EmergencyContact string `json:"emergency_contact" csv:"emergency_contact"`
}

// GenerateEmployeeRecord generates employee PII
func (g *Generator) GenerateEmployeeRecord() *EmployeeRecord {
	base := g.GenerateRecord()
	
	departments := []string{
		"Engineering", "Sales", "Marketing", "HR", "Finance",
		"Operations", "IT", "Legal", "Customer Support",
	}
	
	return &EmployeeRecord{
		Record:         *base,
		EmployeeID:     fmt.Sprintf("EMP%06d", rand.Intn(999999)),
		Department:     departments[rand.Intn(len(departments))],
		JobTitle:       gofakeit.JobTitle(),
		Salary:         rand.Intn(150000) + 40000, // $40k-$190k
		HireDate:       g.generateHireDate(),
		ManagerName:    gofakeit.Name(),
		EmergencyContact: g.generatePhone(),
	}
}

// generateHireDate generates a hire date
func (g *Generator) generateHireDate() string {
	// Generate hire date within last 10 years
	days := rand.Intn(3650)
	hireDate := time.Now().AddDate(0, 0, -days)
	return hireDate.Format("2006-01-02")
}

// MedicalRecord generates healthcare-related PII
type MedicalRecord struct {
	Record
	MRN            string `json:"mrn" csv:"mrn"` // Medical Record Number
	InsuranceID    string `json:"insurance_id" csv:"insurance_id"`
	InsuranceProvider string `json:"insurance_provider" csv:"insurance_provider"`
	PrimaryCarePhysician string `json:"primary_care_physician" csv:"primary_care_physician"`
	BloodType      string `json:"blood_type" csv:"blood_type"`
	Allergies      string `json:"allergies" csv:"allergies"`
	Medications    string `json:"medications" csv:"medications"`
}

// GenerateMedicalRecord generates medical PII
func (g *Generator) GenerateMedicalRecord() *MedicalRecord {
	base := g.GenerateRecord()
	
	insuranceProviders := []string{
		"Blue Cross Blue Shield", "UnitedHealthcare", "Aetna",
		"Cigna", "Humana", "Kaiser Permanente",
	}
	
	bloodTypes := []string{"O+", "O-", "A+", "A-", "B+", "B-", "AB+", "AB-"}
	
	allergies := []string{
		"Penicillin", "Peanuts", "Shellfish", "Latex", "None known",
	}
	
	medications := []string{
		"Lisinopril", "Metformin", "Atorvastatin", "Omeprazole", "None",
	}
	
	return &MedicalRecord{
		Record:            *base,
		MRN:               fmt.Sprintf("MRN%09d", rand.Intn(999999999)),
		InsuranceID:       fmt.Sprintf("INS%010d", rand.Intn(9999999999)),
		InsuranceProvider: insuranceProviders[rand.Intn(len(insuranceProviders))],
		PrimaryCarePhysician: "Dr. " + gofakeit.Name(),
		BloodType:         bloodTypes[rand.Intn(len(bloodTypes))],
		Allergies:         allergies[rand.Intn(len(allergies))],
		Medications:       medications[rand.Intn(len(medications))],
	}
}

// FinancialRecord generates financial PII
type FinancialRecord struct {
	Record
	AccountNumber  string `json:"account_number" csv:"account_number"`
	RoutingNumber  string `json:"routing_number" csv:"routing_number"`
	BankName       string `json:"bank_name" csv:"bank_name"`
	AccountType    string `json:"account_type" csv:"account_type"`
	Balance        float64 `json:"balance" csv:"balance"`
	CreditScore    int    `json:"credit_score" csv:"credit_score"`
	AnnualIncome   int    `json:"annual_income" csv:"annual_income"`
}

// GenerateFinancialRecord generates financial PII
func (g *Generator) GenerateFinancialRecord() *FinancialRecord {
	base := g.GenerateRecord()
	
	banks := []string{
		"Chase", "Bank of America", "Wells Fargo", "Citibank",
		"US Bank", "PNC Bank", "Capital One", "TD Bank",
	}
	
	accountTypes := []string{"Checking", "Savings", "Money Market"}
	
	return &FinancialRecord{
		Record:        *base,
		AccountNumber: fmt.Sprintf("%010d", rand.Intn(9999999999)),
		RoutingNumber: fmt.Sprintf("%09d", rand.Intn(999999999)),
		BankName:      banks[rand.Intn(len(banks))],
		AccountType:   accountTypes[rand.Intn(len(accountTypes))],
		Balance:       float64(rand.Intn(100000)) + rand.Float64(),
		CreditScore:   rand.Intn(400) + 400, // 400-800
		AnnualIncome:  rand.Intn(200000) + 30000, // $30k-$230k
	}
}
