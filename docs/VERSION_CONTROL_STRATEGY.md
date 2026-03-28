# Version Control Strategy - Enhanced Generation

## 🎯 Recommended Approach: Feature Branch with Detailed Commits

This approach provides clear history and easy rollback if needed.

---

## Step-by-Step Git Workflow

### 1. Create Feature Branch

```bash
cd janus-v3
git checkout main  # or master
git pull origin main
git checkout -b feature/enhanced-generation
```

---

### 2. Apply Changes in Order

#### Commit 1: Enhanced Generation Backend
```bash
# Extract backend modules
tar -xzf enhanced-generation-complete.tar.gz

# Stage files
git add internal/core/generator/models/
git add internal/core/generator/validator/
git add internal/core/generator/resolver/
git add internal/core/generator/filler/
git add internal/core/generator/enhanced/

# Commit
git commit -m "feat: Add enhanced generation backend

- Add flexible constraint system (size/count modes)
- Add PII/filler distribution control (10% PII / 90% filler)
- Add comprehensive input validation
- Add disk space safety checks (25% margin or 5GB minimum)
- Add platform-specific implementations (Unix/Windows)
- Support three PII types: standard, healthcare, financial
- Support three formats: csv, json, txt
- Add pre-built scenarios
- Add reproducible generation with seeds
- Add real-time progress tracking

Files:
- models/enhanced.go (204 lines)
- validator/validator.go (396 lines)
- validator/validator_unix.go (52 lines)
- validator/validator_windows.go (82 lines)
- resolver/resolver.go (294 lines)
- filler/filler.go (302 lines)
- enhanced/generator.go (630 lines)
- enhanced/api.go (208 lines)
- enhanced/monitor_unix.go (47 lines)
- enhanced/monitor_windows.go (77 lines)

Total: 2,292 lines across 10 modules"
```

#### Commit 2: CLI Integration
```bash
# Apply CLI changes
cp main-updated.go cmd/janus-cli/main.go

# Stage
git add cmd/janus-cli/main.go

# Commit
git commit -m "feat: Add enhanced generation CLI command

Add 'gen quick' command with comprehensive flag support:
- --total-size: Size-based mode (e.g., 5GB)
- --file-count: Count-based mode (e.g., 10000)
- --pii-percent: PII percentage (0-100)
- --pii-type: standard|healthcare|financial
- --filler-percent: Filler percentage (must total 100)
- --file-size-min/max: File size constraints
- --output: Output directory
- --seed: Reproducible generation

Features:
- Interactive validation with user confirmation
- Disk space checking and warnings
- Generation plan preview
- Clear error messages with suggestions

Usage:
  janus-cli gen quick --file-count 100 --pii-percent 10 --filler-percent 90

Changes:
- Added enhanced, validator, models imports
- Added generateQuick command and handler
- Added validation and confirmation flow"
```

#### Commit 3: Database Driver Fix
```bash
# Apply database changes
cp sqlite-fixed.go internal/database/sqlite/sqlite.go

# Update dependencies
go get modernc.org/sqlite
go mod tidy

# Stage
git add internal/database/sqlite/sqlite.go
git add go.mod go.sum

# Commit
git commit -m "fix: Switch to pure Go SQLite driver

Replace github.com/mattn/go-sqlite3 with modernc.org/sqlite

Benefits:
- No CGO required (no GCC needed on Windows)
- Pure Go implementation
- Cross-platform without compiler dependencies
- Easier build and distribution

Changes:
- Update import to modernc.org/sqlite
- Change driver name from 'sqlite3' to 'sqlite'
- Update go.mod dependencies

Breaking: Requires go get modernc.org/sqlite before building"
```

#### Commit 4: API Handlers Compilation Fix
```bash
# Apply handlers fix
cp handlers-fixed.go internal/api/handlers/handlers.go

# Stage
git add internal/api/handlers/handlers.go

# Commit
git commit -m "fix: Fix unused variable in DestroyScenario handler

Fix compilation error in handlers.go:
- Line 329: Use 'id' variable in response
- Prevents 'id declared and not used' error

Change:
  respondJSON(w, http.StatusOK, map[string]string{
-     \"message\": \"Destruction not yet implemented\",
+     \"scenario_id\": id,
+     \"message\":     \"Destruction not yet implemented\",
  })"
```

#### Commit 5: Server Endpoint (when you add it)
```bash
# After manually adding the route
git add internal/api/handlers/handlers.go
git add cmd/janus-server/main.go  # or wherever routes are registered

# Commit
git commit -m "feat: Add enhanced generation API endpoint

Add POST /api/v1/generate/enhanced endpoint

Features:
- Accepts enhanced generation options via JSON
- Validates options before starting
- Runs generation in background goroutine
- Returns 202 Accepted immediately
- Logs progress to console

Request body:
{
  \"name\": \"Test Generation\",
  \"file_count\": 100,
  \"pii_percent\": 10,
  \"filler_percent\": 90,
  \"output_path\": \"./payloads/test\"
}

Response:
{
  \"status\": \"started\",
  \"message\": \"Generation started in background\",
  \"output_path\": \"./payloads/test\"
}

Changes:
- Added HandleEnhancedGenerate handler
- Registered route in router
- Added enhanced and models imports"
```

#### Commit 6: Documentation
```bash
# Add documentation
git add docs/
git add README.md

# Commit
git commit -m "docs: Add enhanced generation documentation

Add comprehensive documentation:
- User guide with examples
- API endpoint documentation
- CLI command reference
- Pre-built scenarios guide
- Troubleshooting guide
- Performance characteristics

Update README with:
- New features list
- Quick start examples
- Link to documentation"
```

---

### 3. Push and Create Pull Request

```bash
# Push branch
git push origin feature/enhanced-generation

# Create PR via GitHub/GitLab UI or CLI
gh pr create --title "Enhanced Generation System" --body "$(cat PR_TEMPLATE.md)"
```

---

## Pull Request Template

Save as `PR_TEMPLATE.md`:

```markdown
## Enhanced Generation System

### Summary
Adds flexible data generation system with PII/filler distribution control, disk space validation, and cross-platform support.

### Changes
- **Backend**: 10 modules, 2,292 lines of new code
- **CLI**: New `gen quick` command with 10+ flags
- **API**: New `/api/v1/generate/enhanced` endpoint
- **Database**: Switched to pure Go SQLite driver
- **Platform**: Windows, Linux, Mac support

### Features
✅ Volume-based constraints (e.g., generate 5GB of data)
✅ Count-based constraints (e.g., generate 10,000 files)
✅ PII/Filler distribution (10% PII / 90% filler)
✅ Three PII types: standard, healthcare, financial
✅ Three formats: CSV, JSON, TXT
✅ Disk space validation (25% safety margin)
✅ Real-time monitoring during generation
✅ Platform-specific implementations
✅ Reproducible generation with seeds
✅ Pre-built scenarios

### Testing
- [x] Local generation tested (`test_enhanced.go`)
- [x] CLI validation tested
- [x] Disk space checking tested
- [ ] Server endpoint tested (pending route addition)
- [ ] Web UI integration (future work)
- [x] Windows build tested
- [ ] Linux build tested
- [ ] Automated tests (future work)

### Breaking Changes
⚠️ SQLite driver change: Requires `modernc.org/sqlite`
   - Run `go get modernc.org/sqlite` before building
   - Pure Go, no CGO needed

### Documentation
- [x] Backend architecture documented
- [x] CLI usage guide
- [x] API documentation
- [x] Troubleshooting guide
- [ ] Performance benchmarks (future)

### Remaining Tasks
- [ ] Add server route registration
- [ ] Comprehensive testing
- [ ] Web UI integration
- [ ] Performance optimization

### How to Test
```bash
# Local generation (no server)
go run test_enhanced.go

# CLI validation
./janus-cli gen quick --help

# Full flow (requires server endpoint)
./janus-cli gen quick --file-count 10 --pii-percent 50 --filler-percent 50
```

### Screenshots
[Add screenshots of CLI output, generated files]

### Related Issues
Closes #XXX
Implements #YYY
```

---

## Alternative: Squash Strategy

If you prefer cleaner history:

```bash
# Create branch
git checkout -b feature/enhanced-generation

# Apply ALL changes
tar -xzf enhanced-generation-complete.tar.gz
cp main-updated.go cmd/janus-cli/main.go
cp sqlite-fixed.go internal/database/sqlite/sqlite.go
cp handlers-fixed.go internal/api/handlers/handlers.go
go get modernc.org/sqlite
go mod tidy

# Stage everything
git add -A

# Single comprehensive commit
git commit -m "feat: Complete enhanced generation system

Add flexible data generation with PII/filler distribution control.

Backend (2,292 lines across 10 modules):
- Flexible constraints: size mode (5GB) or count mode (10K files)
- PII/Filler distribution: configurable percentages
- Three PII types: standard, healthcare, financial
- Three formats: CSV, JSON, TXT
- Disk validation: 25% safety margin or 5GB minimum
- Platform support: Windows (native), Unix/Linux (syscall)
- Pre-built scenarios: quick-test, realistic-mixed, etc.
- Reproducible: seed support for deterministic generation

CLI Integration:
- New 'gen quick' command with comprehensive flags
- Interactive validation and confirmation
- Disk space warnings
- Generation plan preview

Database:
- Switch to pure Go SQLite driver (modernc.org/sqlite)
- No CGO/GCC required on Windows
- Cross-platform compatibility

API:
- Fix handlers.go compilation errors
- Prepare enhanced generation endpoint (route pending)

Modules:
- models/enhanced.go: Data structures
- validator/: Input and disk validation (Unix/Windows)
- resolver/: Constraint resolution
- filler/: Lorem ipsum generation
- enhanced/: Main orchestrator and API

BREAKING CHANGE: Requires modernc.org/sqlite driver
  Run: go get modernc.org/sqlite

Testing:
  go run test_enhanced.go

See docs/ for complete documentation."

# Push
git push origin feature/enhanced-generation
```

---

## Branch Protection Rules

Recommended settings for main/master branch:

- ✅ Require pull request reviews (1 approval)
- ✅ Require status checks to pass
- ✅ Require branches to be up to date
- ✅ Include administrators
- ✅ Require linear history (optional)

---

## Commit Message Conventions

### Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Tests
- `chore`: Maintenance

### Examples
```
feat(cli): Add gen quick command
fix(handlers): Fix unused variable error
docs(api): Add endpoint documentation
refactor(generator): Extract validation logic
test(enhanced): Add integration tests
chore(deps): Update modernc.org/sqlite
```

---

## Tagging Strategy

After merge:

```bash
git checkout main
git pull origin main

# Create annotated tag
git tag -a v3.0.0 -m "Enhanced Generation Release

Features:
- Enhanced generation system
- CLI gen quick command
- Pure Go SQLite driver
- Platform support (Windows/Unix)

Breaking: Requires modernc.org/sqlite"

# Push tag
git push origin v3.0.0
```

---

## Summary

**Recommended:** Feature branch with 6 detailed commits
**Alternative:** Single squash commit for cleaner history
**After Merge:** Tag as v3.0.0 or similar

**Benefits of Detailed Commits:**
- Clear history of what changed when
- Easy to cherry-pick specific changes
- Better for code review
- Easier to debug issues

**Benefits of Squash:**
- Cleaner main branch history
- Simpler to understand at high level
- Easier to revert entire feature

**Choose based on your team's preference!**
