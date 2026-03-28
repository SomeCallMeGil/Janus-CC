# Enhanced Data Generation - Integration Guide

## Quick Integration Steps

### Step 1: Extract the Code

```bash
# Extract to your janus-v3 project
cd janus-v3
tar -xzf enhanced-generation-complete.tar.gz

# Verify structure
ls -la internal/core/generator/
# Should see: models/ validator/ resolver/ filler/ enhanced/
```

### Step 2: Test the Backend

```go
// test_enhanced.go
package main

import (
    "fmt"
    "janus/internal/core/generator/enhanced"
    "janus/internal/database/sqlite"
)

func main() {
    // Setup
    db, _ := sqlite.New("./test.db")
    db.Connect()
    db.Migrate()
    
    // Quick test
    opts := enhanced.QuickGenerateOptions{
        Name:          "Backend Test",
        OutputPath:    "./test-payload",
        FileCount:     10,  // Just 10 files for testing
        FileSizeMin:   "1KB",
        FileSizeMax:   "10KB",
        PIIPercent:    50,
        PIIType:       "standard",
        FillerPercent: 50,
        Formats:       []string{"csv", "json", "txt"},
    }
    
    fmt.Println("Testing enhanced generation...")
    result, err := enhanced.QuickGenerate(db, opts, func(p enhanced.Progress) {
        fmt.Printf("\rProgress: %d/%d", p.Current, p.Total)
    })
    
    if err != nil {
        fmt.Printf("\n❌ Error: %v\n", err)
        return
    }
    
    fmt.Printf("\n✅ Success! Created %d files\n", result.FilesCreated)
}
```

```bash
# Run test
go run test_enhanced.go

# Expected output:
# Testing enhanced generation...
# Progress: 10/10
# ✅ Success! Created 10 files

# Verify files created
ls -lh test-payload/
```

### Step 3: Add to Existing CLI (Option A - Quick)

```go
// In cmd/janus-cli/main.go

import "janus/internal/core/generator/enhanced"

// Add new command
generateQuickCmd := &cobra.Command{
    Use:   "quick",
    Short: "Quick generation with enhanced features",
    RunE:  generateQuick,
}

generateQuickCmd.Flags().String("total-size", "", "Total size (e.g., 5GB)")
generateQuickCmd.Flags().Int("file-count", 0, "Number of files")
generateQuickCmd.Flags().String("file-size-min", "1KB", "Min file size")
generateQuickCmd.Flags().String("file-size-max", "10MB", "Max file size")
generateQuickCmd.Flags().Float64("pii-percent", 10, "PII percentage (0-100)")
generateQuickCmd.Flags().String("pii-type", "standard", "PII type (standard/healthcare/financial)")
generateQuickCmd.Flags().Float64("filler-percent", 90, "Filler percentage (0-100)")
generateQuickCmd.Flags().String("output", "./payloads/quick", "Output directory")

rootCmd.AddCommand(generateQuickCmd)

func generateQuick(cmd *cobra.Command, args []string) error {
    // Get flags
    totalSize, _ := cmd.Flags().GetString("total-size")
    fileCount, _ := cmd.Flags().GetInt("file-count")
    fileSizeMin, _ := cmd.Flags().GetString("file-size-min")
    fileSizeMax, _ := cmd.Flags().GetString("file-size-max")
    piiPercent, _ := cmd.Flags().GetFloat64("pii-percent")
    piiType, _ := cmd.Flags().GetString("pii-type")
    fillerPercent, _ := cmd.Flags().GetFloat64("filler-percent")
    output, _ := cmd.Flags().GetString("output")
    
    // Build options
    opts := enhanced.QuickGenerateOptions{
        Name:          "CLI Generated",
        OutputPath:    output,
        TotalSize:     totalSize,
        FileCount:     fileCount,
        FileSizeMin:   fileSizeMin,
        FileSizeMax:   fileSizeMax,
        PIIPercent:    piiPercent,
        PIIType:       piiType,
        FillerPercent: fillerPercent,
        Formats:       []string{"csv", "json", "txt"},
    }
    
    // Generate
    result, err := enhanced.QuickGenerate(db, opts, func(p enhanced.Progress) {
        fmt.Printf("\r[%s] %d/%d (%.1f%%)    ", 
            p.Status, p.Current, p.Total, p.Percent)
    })
    
    if err != nil {
        return err
    }
    
    fmt.Printf("\n✅ Generated %d files in %s\n", 
        result.FilesCreated, result.OutputPath)
    
    return nil
}
```

**Usage:**
```bash
# Size mode
./janus-cli generate quick --total-size 5GB --pii-percent 10 --pii-type healthcare --filler-percent 90

# Count mode  
./janus-cli generate quick --file-count 10000 --pii-percent 25 --pii-type standard --filler-percent 75

# With custom output
./janus-cli generate quick --total-size 1GB --output ./my-test --pii-percent 15 --filler-percent 85
```

### Step 4: Add to API (Option B)

```go
// In internal/api/handlers/handlers.go

import "janus/internal/core/generator/enhanced"

// Add handler
func (h *Handlers) HandleEnhancedGenerate(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name          string   `json:"name"`
        OutputPath    string   `json:"output_path"`
        TotalSize     string   `json:"total_size"`
        FileCount     int      `json:"file_count"`
        FileSizeMin   string   `json:"file_size_min"`
        FileSizeMax   string   `json:"file_size_max"`
        PIIPercent    float64  `json:"pii_percent"`
        PIIType       string   `json:"pii_type"`
        FillerPercent float64  `json:"filler_percent"`
        Formats       []string `json:"formats"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondJSON(w, 400, map[string]string{"error": err.Error()})
        return
    }
    
    // Convert to options
    opts := enhanced.QuickGenerateOptions{
        Name:          req.Name,
        OutputPath:    req.OutputPath,
        TotalSize:     req.TotalSize,
        FileCount:     req.FileCount,
        FileSizeMin:   req.FileSizeMin,
        FileSizeMax:   req.FileSizeMax,
        PIIPercent:    req.PIIPercent,
        PIIType:       req.PIIType,
        FillerPercent: req.FillerPercent,
        Formats:       req.Formats,
    }
    
    // Validate first
    if err := enhanced.Validate(h.db, opts); err != nil {
        respondJSON(w, 400, map[string]string{"error": err.Error()})
        return
    }
    
    // Generate in background
    go func() {
        result, err := enhanced.QuickGenerate(h.db, opts, func(p enhanced.Progress) {
            // Broadcast progress via WebSocket
            h.hub.Broadcast(map[string]interface{}{
                "type":    "generation_progress",
                "current": p.Current,
                "total":   p.Total,
                "percent": p.Percent,
                "status":  p.Status,
            })
        })
        
        if err != nil {
            h.hub.Broadcast(map[string]interface{}{
                "type":  "generation_error",
                "error": err.Error(),
            })
        } else {
            h.hub.Broadcast(map[string]interface{}{
                "type":         "generation_complete",
                "files_created": result.FilesCreated,
                "bytes_written": result.BytesWritten,
                "duration":     result.Duration.String(),
            })
        }
    }()
    
    respondJSON(w, 202, map[string]string{"status": "started"})
}

// In internal/api/server.go, add route:
r.Post("/api/v1/generate/enhanced", h.HandleEnhancedGenerate)
```

**API Usage:**
```bash
curl -X POST http://localhost:8080/api/v1/generate/enhanced \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Test",
    "output_path": "./payloads/api-test",
    "total_size": "1GB",
    "file_size_min": "1KB",
    "file_size_max": "10MB",
    "pii_percent": 20,
    "pii_type": "healthcare",
    "filler_percent": 80,
    "formats": ["csv", "json"]
  }'
```

### Step 5: Add Pre-built Scenarios

```go
// CLI: Add shortcut command
prebuiltCmd := &cobra.Command{
    Use:   "prebuilt [scenario-name]",
    Short: "Use a pre-built scenario",
    Args:  cobra.ExactArgs(1),
    RunE:  usePrebuilt,
}

func usePrebuilt(cmd *cobra.Command, args []string) error {
    scenarioName := args[0]
    
    opts, exists := enhanced.GetPrebuiltScenario(scenarioName)
    if !exists {
        fmt.Printf("Unknown scenario. Available:\n")
        for _, name := range enhanced.ListPrebuiltScenarios() {
            fmt.Printf("  - %s\n", name)
        }
        return fmt.Errorf("scenario not found: %s", scenarioName)
    }
    
    fmt.Printf("Using pre-built scenario: %s\n\n", scenarioName)
    
    result, err := enhanced.QuickGenerate(db, opts, progressCallback)
    // ...
}
```

**Usage:**
```bash
# List scenarios
./janus-cli generate prebuilt --list

# Use one
./janus-cli generate prebuilt healthcare-large
./janus-cli generate prebuilt quick-pii-test
./janus-cli generate prebuilt mixed-realistic
```

---

## Common Integration Patterns

### Pattern 1: CLI with Progress Bar

```go
import "github.com/schollz/progressbar/v3"

func generateWithProgressBar(opts enhanced.QuickGenerateOptions) error {
    var bar *progressbar.ProgressBar
    
    result, err := enhanced.QuickGenerate(db, opts, func(p enhanced.Progress) {
        if bar == nil && p.Total > 0 {
            bar = progressbar.Default(int64(p.Total))
        }
        if bar != nil {
            bar.Set(p.Current)
        }
    })
    
    return err
}
```

### Pattern 2: API with WebSocket

```javascript
// Frontend
const ws = new WebSocket('ws://localhost:8080/ws/v1/activity');

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    if (data.type === 'generation_progress') {
        updateProgressBar(data.percent);
        updateStatus(`${data.current}/${data.total} files`);
    }
    
    if (data.type === 'generation_complete') {
        showSuccess(`Created ${data.files_created} files!`);
    }
    
    if (data.type === 'generation_error') {
        showError(data.error);
    }
};

// Start generation
fetch('/api/v1/generate/enhanced', {
    method: 'POST',
    body: JSON.stringify(generationOptions)
});
```

### Pattern 3: Validation Before Generate

```go
func smartGenerate(opts enhanced.QuickGenerateOptions) error {
    // Validate first
    fmt.Println("🔍 Validating...")
    if err := enhanced.Validate(db, opts); err != nil {
        return err
    }
    
    // Ask for confirmation on warnings
    fmt.Print("\nProceed with generation? [Y/n]: ")
    var response string
    fmt.Scanln(&response)
    
    if response == "n" || response == "N" {
        return nil
    }
    
    // Generate
    fmt.Println("\n🚀 Generating...")
    return enhanced.QuickGenerate(db, opts, progressCallback)
}
```

---

## Troubleshooting

### Issue 1: Import Errors

```
Error: cannot find package "janus/internal/core/generator/enhanced"
```

**Fix:**
```bash
# Ensure go.mod is in the right place
cd janus-v3
go mod tidy

# Verify structure
ls internal/core/generator/enhanced/
# Should show: generator.go, api.go
```

### Issue 2: Compilation Errors

```
Error: undefined: models.ParseSize
```

**Fix:** Import the models package
```go
import "janus/internal/core/generator/models"
```

### Issue 3: Disk Space Errors

```
Error: disk space validation failed: Insufficient disk space
```

**Fix:** This is working correctly! Free up space or reduce generation size.

---

## Testing the Integration

### Test 1: Basic Generation
```bash
./janus-cli generate quick \
  --file-count 10 \
  --file-size-min 1KB \
  --file-size-max 10KB \
  --pii-percent 100 \
  --pii-type standard \
  --filler-percent 0

# Should create 10 small PII files
```

### Test 2: Large Generation
```bash
./janus-cli generate quick \
  --total-size 100MB \
  --file-size-min 100KB \
  --file-size-max 1MB \
  --pii-percent 10 \
  --pii-type healthcare \
  --filler-percent 90

# Should create ~100 files totaling ~100MB
```

### Test 3: Disk Space Warning
```bash
# On a nearly-full disk
./janus-cli generate quick --total-size 100GB

# Should show warning or error about disk space
```

---

## Next Steps

1. **Integrate with CLI** (2 hours)
   - Add `generate quick` command
   - Add pre-built shortcuts
   - Pretty progress output

2. **Integrate with API** (1 hour)
   - Add enhanced generation endpoint
   - Background processing
   - WebSocket progress

3. **Build Web UI** (2-3 hours)
   - Visual form builder
   - Real-time validation
   - Progress visualization

---

## Backend Complete ✅

You now have:
- ✅ Smart constraint system
- ✅ Comprehensive validation
- ✅ Disk space safety
- ✅ Filler data generation
- ✅ Progress monitoring
- ✅ Error handling
- ✅ Easy integration

**Ready to build the frontend!**
