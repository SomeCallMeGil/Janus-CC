# Enhanced Generation - Usage Examples

## Quick Start

### Example 1: Generate 5GB of Mixed Data (10% PII, 90% Filler)

```go
package main

import (
	"fmt"
	
	"janus/internal/core/generator/enhanced"
	"janus/internal/database/sqlite"
)

func main() {
	// Initialize database
	db, _ := sqlite.New("./janus.db")
	db.Connect()
	
	// Create quick options
	opts := enhanced.QuickGenerateOptions{
		Name:       "Test 5GB",
		OutputPath: "./my-payload",
		
		// Primary constraint: Total size
		TotalSize: "5GB",  // System calculates ~500 files automatically
		
		// File size range
		FileSizeMin: "1KB",
		FileSizeMax: "10MB",
		
		// Distribution (must total 100%)
		PIIPercent:    10.0,  // 10% real healthcare data
		PIIType:       "healthcare",
		FillerPercent: 90.0,  // 90% lorem ipsum
		
		// Formats
		Formats: []string{"csv", "json", "txt"},
		
		// Optional
		DirectoryDepth: 3,
		Seed:           12345, // Reproducible generation
	}
	
	// Generate with progress callback
	result, err := enhanced.QuickGenerate(db, opts, func(progress enhanced.Progress) {
		fmt.Printf("\r[%s] %d/%d files (%.1f%%) - %s",
			progress.Status,
			progress.Current,
			progress.Total,
			progress.Percent,
			progress.CurrentFile)
	})
	
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}
	
	fmt.Printf("\n✅ Complete!\n")
	fmt.Printf("   Files created: %d\n", result.FilesCreated)
	fmt.Printf("   Bytes written: %s\n", formatBytes(result.BytesWritten))
	fmt.Printf("   Duration: %s\n", result.Duration)
}
```

**Output:**
```
📋 Generation Plan:
  Output: ./my-payload
  Total Files: 523
    • PII (healthcare): 52 files
    • Filler: 471 files
  Estimated Size: 5 GB
  File Size Range: 1 KB - 10 MB
  Directory Depth: 3 levels
  Formats: [csv json txt]
  Seed: 12345 (reproducible: true)

💾 Disk Space:
  Available: 245 GB (after 50 GB safety margin)
  Will use: 5 GB (2.0% of available)
  Remaining: 240 GB

[generating] 523/523 files (100.0%) - pii_1708531234_0522.csv

✅ Complete!
   Files created: 523
   Bytes written: 5.0 GB
   Duration: 2m 15s
```

---

### Example 2: Generate Exactly 10,000 Files

```go
opts := enhanced.QuickGenerateOptions{
	Name:       "10K Files",
	OutputPath: "./payload-10k",
	
	// Primary constraint: File count
	FileCount: 10000,  // System calculates sizes randomly
	
	// File size range (random per file)
	FileSizeMin: "500B",
	FileSizeMax: "5MB",
	
	// 25% PII, 75% filler
	PIIPercent:    25.0,
	PIIType:       "standard",
	FillerPercent: 75.0,
	
	Formats: []string{"csv", "json"},
}

result, _ := enhanced.QuickGenerate(db, opts, nil) // No progress callback
```

---

### Example 3: Use Pre-built Scenarios

```go
// List available scenarios
scenarios := enhanced.ListPrebuiltScenarios()
fmt.Println("Available scenarios:", scenarios)

// Get pre-built scenario
opts, exists := enhanced.GetPrebuiltScenario("healthcare-large")
if exists {
	result, _ := enhanced.QuickGenerate(db, opts, progressCallback)
}

// Pre-built scenarios:
// - "quick-pii-test": 1,000 files, 100% PII
// - "mixed-realistic": 1GB, 15% PII / 85% filler
// - "healthcare-large": 5GB, 30% healthcare / 70% filler
// - "financial-test": 5,000 files, 40% financial / 60% filler
```

---

### Example 4: Pre-validate Before Generating

```go
opts := enhanced.QuickGenerateOptions{
	Name:          "Big Test",
	OutputPath:    "/small-disk",
	TotalSize:     "100GB",  // Too big!
	FileSizeMin:   "1MB",
	FileSizeMax:   "10MB",
	PIIPercent:    50,
	PIIType:       "standard",
	FillerPercent: 50,
}

// Validate first (doesn't generate)
err := enhanced.Validate(db, opts)
if err != nil {
	fmt.Printf("Validation failed:\n%v\n", err)
	// Output:
	// Validation failed:
	// • disk_space: Insufficient disk space.
	//   Required: 100 GB
	//   Available: 45 GB (after 25 GB safety margin)
	return
}

// If validation passes, then generate
result, _ := enhanced.QuickGenerate(db, opts, nil)
```

---

### Example 5: Handle Errors Gracefully

```go
opts := enhanced.QuickGenerateOptions{
	Name:          "Test",
	OutputPath:    "./payload",
	TotalSize:     "5GB",
	FileSizeMin:   "10MB",  // ERROR: Min > Max!
	FileSizeMax:   "1MB",
	PIIPercent:    60,      // ERROR: Doesn't total 100%
	PIIType:       "standard",
	FillerPercent: 50,
}

result, err := enhanced.QuickGenerate(db, opts, nil)
if err != nil {
	fmt.Printf("❌ Generation failed:\n%v\n", err)
	// Output:
	// ❌ Generation failed:
	// validation failed:
	// • file_size: Minimum file size (10 MB) cannot be greater than maximum (1 MB)
	// • distribution: Distribution percentages must total 100% (currently 110.0%)
	return
}
```

---

## Advanced Usage

### Custom Progress Monitoring

```go
func main() {
	startTime := time.Now()
	
	result, _ := enhanced.QuickGenerate(db, opts, func(progress enhanced.Progress) {
		switch progress.Status {
		case "validating":
			fmt.Println("🔍 Validating options...")
			
		case "planning":
			fmt.Println("📋 Creating generation plan...")
			
		case "generating":
			// Calculate rate
			elapsed := time.Since(startTime).Seconds()
			rate := float64(progress.Current) / elapsed
			
			// Estimate remaining
			remaining := float64(progress.Total - progress.Current) / rate
			
			fmt.Printf("\r📝 Generating: %d/%d (%.1f%%) - %.1f files/sec - ETA: %.0fs",
				progress.Current,
				progress.Total,
				progress.Percent,
				rate,
				remaining)
			
		case "complete":
			fmt.Println("\n✅ Generation complete!")
			
		case "error":
			fmt.Println("\n❌ Generation failed")
		}
	})
}
```

---

### Integration with Encryption

```go
// Generate files
result, _ := enhanced.QuickGenerate(db, genOpts, nil)

fmt.Printf("✅ Generated %d files in %s\n", 
	result.FilesCreated, 
	result.OutputPath)

// Now encrypt them
encryptor := encryptor.New(db, "password", 4)

encryptOpts := encryptor.DefaultOptions()
encryptOpts.Mode = encryptor.ModePartialEncryption

fmt.Println("🔐 Encrypting 25% of files...")

// Encrypt 25%
err := encryptor.EncryptScenario(result.ScenarioID, 25.0, encryptOpts, func(r encryptor.EncryptResult) {
	if r.Error == nil {
		fmt.Printf("✓ Encrypted: %s\n", r.FilePath)
	}
})

fmt.Println("✅ Done! Ready for testing.")
```

---

## Validation Rules

### Input Validation
- ✅ `FileSizeMin` cannot be greater than `FileSizeMax`
- ✅ `PIIPercent + FillerPercent` must equal 100.0
- ✅ All percentages must be 0-100
- ✅ Must specify either `TotalSize` OR `FileCount` (not both)
- ✅ At least one format must be selected
- ✅ PIIType must be: "standard", "healthcare", or "financial"

### Disk Space Validation
- ✅ Pre-flight check before generation
- ✅ 25% of disk must remain free (or 5GB minimum, whichever is larger)
- ✅ Real-time monitoring during generation
- ✅ Emergency stop if space gets critical

### Warnings (Don't Block)
- ⚠️ Very large file sizes (>10GB per file)
- ⚠️ Very large total sizes (>1TB)
- ⚠️ Very many files (>1 million)
- ⚠️ Disk space is tight (less than 2x safety margin)
- ⚠️ Disk is >80% full

---

## Error Handling

```go
result, err := enhanced.QuickGenerate(db, opts, callback)
if err != nil {
	// Generation failed - could be:
	// - Validation error (bad inputs)
	// - Disk space error (not enough room)
	// - Disk full during generation (emergency stop)
	// - File system error (permissions, etc.)
	
	fmt.Printf("Error: %v\n", err)
	return
}

// Check for partial success
if !result.Success {
	fmt.Printf("⚠️ Partial failure:\n")
	fmt.Printf("   Created: %d files\n", result.FilesCreated)
	fmt.Printf("   Errors: %d\n", len(result.Errors))
	
	for _, err := range result.Errors {
		fmt.Printf("   - %v\n", err)
	}
}
```

---

## Best Practices

1. **Always validate first** for large generations
2. **Use progress callbacks** for long-running operations
3. **Set a seed** for reproducible test data
4. **Start small** - test with smaller sizes first
5. **Check disk space** before large generations
6. **Use pre-built scenarios** for common use cases
7. **Monitor output** - watch for warnings
8. **Clean up** after testing (large directories)

---

## Performance Tips

- **Larger files = fewer files = faster** (less file system overhead)
- **Fewer formats = faster** (only use formats you need)
- **Lower directory depth = faster** (less mkdir operations)
- **SSD > HDD** for many small files
- **Use seed=0 for production** (slightly faster than seeded)

---

## Common Patterns

### Pattern 1: Quick Test
```go
// Just need some test files fast
opts := enhanced.QuickGenerateOptions{
	Name:          "Quick Test",
	OutputPath:    "./test",
	FileCount:     100,
	FileSizeMin:   "1KB",
	FileSizeMax:   "100KB",
	PIIPercent:    100,
	PIIType:       "standard",
	FillerPercent: 0,
}
```

### Pattern 2: Realistic Environment
```go
// Simulate real environment
opts := enhanced.QuickGenerateOptions{
	Name:          "Realistic",
	OutputPath:    "./realistic",
	TotalSize:     "10GB",
	FileSizeMin:   "100B",
	FileSizeMax:   "100MB",
	PIIPercent:    5,    // 5% sensitive data (realistic)
	PIIType:       "standard",
	FillerPercent: 95,   // 95% normal files
}
```

### Pattern 3: Stress Test
```go
// Test with many files
opts := enhanced.QuickGenerateOptions{
	Name:          "Stress Test",
	OutputPath:    "./stress",
	FileCount:     100000,  // 100K files
	FileSizeMin:   "1B",
	FileSizeMax:   "10KB",
	PIIPercent:    10,
	PIIType:       "standard",
	FillerPercent: 90,
}
```

---

Ready to use! Just copy these examples into your code.
