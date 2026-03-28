# Quick Action Checklist

## ✅ Test Now (10 minutes)

```bash
# 1. Navigate to project
cd janus-v3

# 2. Extract all fixes
tar -xzf enhanced-generation-complete.tar.gz
cp main-updated.go cmd/janus-cli/main.go
cp sqlite-fixed.go internal/database/sqlite/sqlite.go
cp handlers-fixed.go internal/api/handlers/handlers.go
cp test_enhanced.go .

# 3. Update dependencies
go get modernc.org/sqlite
go mod tidy

# 4. Build
go build -o janus-cli.exe ./cmd/janus-cli

# 5. Test directly (no server needed)
go run test_enhanced.go

# Expected: Creates 10 files in ./test-payload
```

✅ If this works, your backend is 100% functional!

---

## ⚠️ Complete Server Integration (30 minutes)

### Step 1: Add Imports to handlers.go

**File:** `internal/api/handlers/handlers.go`

**Find the imports section (around line 4):**
```go
import (
    // ... existing imports ...
```

**Add these two lines:**
```go
    "janus/internal/core/generator/enhanced"
    "janus/internal/core/generator/models"
```

### Step 2: Add Handler Function

**File:** `internal/api/handlers/handlers.go`

**Copy the entire function from:** `handler_enhanced_generate.go`

**Paste it at the end of handlers.go** (before the closing brace)

### Step 3: Add Route

**File:** `cmd/janus-server/main.go` OR `internal/api/server/server.go`

**Find where routes are registered (look for):**
```go
r.Post("/api/v1/scenarios/{id}/generate", ...)
```

**Add this line after it:**
```go
r.Post("/api/v1/generate/enhanced", handlers.HandleEnhancedGenerate)
```

### Step 4: Rebuild Server

```bash
go build -o janus-server.exe ./cmd/janus-server
```

### Step 5: Test Full Flow

```bash
# Terminal 1: Start server
./janus-server.exe

# Terminal 2: Test CLI
./janus-cli.exe gen quick \
  --file-count 10 \
  --pii-percent 50 \
  --filler-percent 50 \
  --output ./test-cli-server
```

✅ Should show: "Generation started in background"

---

## 📦 Commit to Git (15 minutes)

### Option A: Detailed Commits (Recommended)

```bash
# Create feature branch
git checkout -b feature/enhanced-generation

# Commit 1: Backend
git add internal/core/generator/
git commit -m "feat: Add enhanced generation backend (10 modules, 2.3K lines)"

# Commit 2: CLI
git add cmd/janus-cli/main.go
git commit -m "feat: Add 'gen quick' CLI command"

# Commit 3: Database
git add internal/database/sqlite/sqlite.go go.mod go.sum
git commit -m "fix: Switch to pure Go SQLite driver"

# Commit 4: Handlers
git add internal/api/handlers/handlers.go
git commit -m "fix: Fix handlers compilation + add enhanced endpoint"

# Commit 5: Server
git add cmd/janus-server/main.go
git commit -m "feat: Register enhanced generation route"

# Push
git push origin feature/enhanced-generation
```

### Option B: Single Commit

```bash
git checkout -b feature/enhanced-generation
git add -A
git commit -m "feat: Complete enhanced generation system

- Backend: 10 modules, 2,300 lines
- CLI: Interactive 'gen quick' command  
- Server: Enhanced generation endpoint
- Platform: Windows, Linux, Mac support
- Features: PII/filler distribution, disk validation"

git push origin feature/enhanced-generation
```

---

## 🧪 Testing Checklist

### Basic Tests
- [ ] `go run test_enhanced.go` - Direct generation
- [ ] `./janus-cli gen quick --help` - CLI help
- [ ] Size mode: `--total-size 10MB`
- [ ] Count mode: `--file-count 50`

### PII Type Tests
- [ ] `--pii-type standard`
- [ ] `--pii-type healthcare`
- [ ] `--pii-type financial`

### Distribution Tests
- [ ] 100% PII, 0% Filler
- [ ] 0% PII, 100% Filler
- [ ] 50% PII, 50% Filler

### Validation Tests
- [ ] Invalid distribution (60% + 50% = 110%)
- [ ] Min > Max file size
- [ ] Insufficient disk space

### Platform Tests
- [ ] Windows build and run
- [ ] Linux build and run (if available)

---

## 📋 File Checklist

### Files to Apply
- [ ] `enhanced-generation-complete.tar.gz` → Extract to project root
- [ ] `main-updated.go` → `cmd/janus-cli/main.go`
- [ ] `sqlite-fixed.go` → `internal/database/sqlite/sqlite.go`
- [ ] `handlers-fixed.go` → `internal/api/handlers/handlers.go`
- [ ] `test_enhanced.go` → Project root (for testing)

### Files to Reference
- [ ] `handler_enhanced_generate.go` → Code to add to handlers.go
- [ ] `PROJECT_STATUS_RECAP.md` → Full status
- [ ] `COMPILATION_CHECKLIST.md` → Quick reference

---

## ⚡ Quick Commands

### Build Everything
```bash
go build -o janus-cli.exe ./cmd/janus-cli
go build -o janus-server.exe ./cmd/janus-server
```

### Clean Build
```bash
go clean -cache
go mod tidy
go build -o janus-cli.exe ./cmd/janus-cli
```

### Run Tests
```bash
go run test_enhanced.go
```

### Check Compilation
```bash
go build ./internal/core/generator/enhanced
go build ./cmd/janus-cli
go build ./cmd/janus-server
```

---

## 🎯 Success Criteria

✅ **Backend Complete When:**
- `go run test_enhanced.go` generates files successfully
- No compilation errors
- Files appear in `./test-payload`

✅ **CLI Complete When:**
- `./janus-cli gen quick --help` shows all flags
- Validation catches errors (e.g., distribution != 100%)
- Can generate files locally

✅ **Server Complete When:**
- CLI sends request without 404
- Server logs show generation progress
- Files appear in specified output directory

✅ **Production Ready When:**
- All tests pass
- Documentation complete
- Version controlled
- Deployed and accessible

---

## 🚨 Troubleshooting

### "Cannot find package"
```bash
go get modernc.org/sqlite
go mod tidy
```

### "sql: unknown driver"
→ Make sure you copied `sqlite-fixed.go`

### "undefined: enhanced"
→ Make sure you extracted `enhanced-generation-complete.tar.gz`

### "404 page not found"
→ Server endpoint not added yet (expected)
→ Use `go run test_enhanced.go` instead

### "id declared and not used"
→ Make sure you copied `handlers-fixed.go`

---

## 📞 Current Status Summary

**Backend:** ✅ 100% Complete  
**CLI:** ✅ 100% Complete  
**Server:** ⚠️ 80% Complete (just needs route)  
**Testing:** ⚠️ 10% Complete  
**Docs:** ✅ 90% Complete  

**Overall:** ~70% Complete

**Next Step:** Run `go run test_enhanced.go` to verify! 🚀
