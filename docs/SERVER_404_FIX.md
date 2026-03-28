# Enhanced Generation Server Endpoint - 404 Fix

## The 404 Error

Your CLI is working correctly! It's trying to call:
```
POST http://localhost:8080/api/v1/generate/enhanced
```

But this endpoint doesn't exist on the server yet.

---

## Quick Test (Option 1 - Recommended)

Test the generation **locally** without the server using this script:

### Create: `test_enhanced.go`

```go
package main

import (
	"fmt"
	
	"janus/internal/core/generator/enhanced"
	"janus/internal/database/sqlite"
)

func main() {
	// Setup database
	db, _ := sqlite.New("./test.db")
	db.Connect()
	db.Migrate()
	
	// Configure generation
	opts := enhanced.QuickGenerateOptions{
		Name:          "Direct Test",
		OutputPath:    "./test-payload",
		FileCount:     10,
		FileSizeMin:   "1KB",
		FileSizeMax:   "10KB",
		PIIPercent:    50,
		PIIType:       "standard",
		FillerPercent: 50,
		Formats:       []string{"csv", "json", "txt"},
	}
	
	fmt.Println("🚀 Testing enhanced generation...")
	
	result, err := enhanced.QuickGenerate(db, opts, func(p enhanced.Progress) {
		fmt.Printf("\r[%s] %d/%d files (%.1f%%)    ",
			p.Status, p.Current, p.Total, p.Percent)
	})
	
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		return
	}
	
	fmt.Printf("\n✅ Success!\n")
	fmt.Printf("   Files: %d\n", result.FilesCreated)
	fmt.Printf("   Size: %d bytes\n", result.BytesWritten)
	fmt.Printf("   Duration: %s\n", result.Duration)
	fmt.Printf("   Location: %s\n", result.OutputPath)
}
```

### Run it:
```bash
go run test_enhanced.go
```

This will generate files **directly** without needing the server!

---

## Add Server Endpoint (Option 2 - Complete Solution)

### Step 1: Add Handler Function

**File:** `internal/api/handlers/handlers.go`

**Add these imports at the top:**
```go
import (
	// ... existing imports ...
	"janus/internal/core/generator/enhanced"
	"janus/internal/core/generator/models"
)
```

**Add this function (after your other handlers):**
```go
// HandleEnhancedGenerate handles enhanced generation requests
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
		Seed          int      `json:"seed"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Build enhanced options
	opts := enhanced.QuickGenerateOptions{
		Name:           req.Name,
		OutputPath:     req.OutputPath,
		TotalSize:      req.TotalSize,
		FileCount:      req.FileCount,
		FileSizeMin:    req.FileSizeMin,
		FileSizeMax:    req.FileSizeMax,
		PIIPercent:     req.PIIPercent,
		PIIType:        req.PIIType,
		FillerPercent:  req.FillerPercent,
		Formats:        req.Formats,
		DirectoryDepth: 3,
		Seed:           int64(req.Seed),
	}
	
	// Validate first
	if err := enhanced.Validate(h.db, opts); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Generate in background
	go func() {
		result, err := enhanced.QuickGenerate(h.db, opts, nil)
		if err != nil {
			// Log error
			fmt.Printf("❌ Generation failed: %v\n", err)
			return
		}
		
		fmt.Printf("✅ Generated %d files (%s) in %s\n",
			result.FilesCreated,
			models.FormatBytes(result.BytesWritten),
			result.Duration)
	}()
	
	respondJSON(w, http.StatusAccepted, map[string]string{
		"status":  "started",
		"message": "Generation started in background",
	})
}
```

### Step 2: Add Route

**Find your router setup file** (likely `cmd/janus-server/main.go` or `internal/api/server/server.go`)

**Find where routes are defined:**
```go
r.Post("/api/v1/scenarios/{id}/generate", handlers.GenerateScenario)
r.Post("/api/v1/scenarios/{id}/encrypt", handlers.EncryptScenario)
```

**Add this line:**
```go
r.Post("/api/v1/generate/enhanced", handlers.HandleEnhancedGenerate)
```

### Step 3: Restart Server

```bash
# Rebuild and run server
go build -o janus-server.exe ./cmd/janus-server
./janus-server.exe
```

### Step 4: Test CLI Again

```bash
./janus-cli.exe gen quick \
  --file-count 10 \
  --pii-percent 50 \
  --pii-type standard \
  --filler-percent 50 \
  --output ./test-payload
```

Should work now! ✅

---

## What You're Seeing Now

```bash
./janus-cli.exe gen quick --file-count 10 --pii-percent 50 --filler-percent 50

🔍 Validating options...

📋 Generation Plan:
  Mode: Count-constrained (10 files)
  Output: ./payloads/quick
  File Size: 1KB - 10MB
  Distribution:
    • PII (standard): 50%
    • Filler: 50%
  Formats: csv, json, txt

💾 Disk Space:
  Available: 245 GB (after safety margin)
  Will use: ~55 KB

Start generation? [Y/n]: y

🚀 Starting generation...
(Sending request to server...)
❌ Error: server error (404): 404 page not found
```

**This is correct!** The CLI validation works perfectly. You just need to add the server endpoint.

---

## Recommended Approach

### For Testing Now:
1. Use the `test_enhanced.go` script (Option 1)
2. This tests everything end-to-end locally
3. No server needed

### For Production:
1. Add the server endpoint (Option 2)
2. CLI and server work together
3. Full API integration

---

## Quick Comparison

| Method | Pros | Cons |
|--------|------|------|
| **test_enhanced.go** | ✅ Tests everything<br>✅ No server needed<br>✅ Quick to test | ❌ Not integrated with CLI<br>❌ Manual script |
| **Server endpoint** | ✅ Full integration<br>✅ CLI works fully<br>✅ Production-ready | ❌ Requires server changes<br>❌ More setup |

---

## Summary

**The 404 is expected** because the server endpoint doesn't exist yet.

**Two solutions:**
1. **Quick test:** Use `test_enhanced.go` to verify generation works
2. **Full solution:** Add server endpoint for CLI integration

**What's working:**
- ✅ CLI validation
- ✅ Disk space checking
- ✅ Input validation
- ✅ Generation plan display
- ❌ Server endpoint (not implemented yet)

Want me to create the complete server integration with the endpoint added?
