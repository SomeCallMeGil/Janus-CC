# SQLite Driver Error - "unknown driver sqlite3"

## The Error
```
sql: unknown driver "sqlite3" (forgotten import?)
```

## Root Cause

The SQLite driver needs to be:
1. Imported (even if not directly used)
2. The correct driver for your platform

---

## Solution: Use Pure Go SQLite Driver

### Step 1: Update Database Connection File

**File:** `internal/database/sqlite/sqlite.go`

**Find line ~9:**
```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"  // ← OLD (requires GCC on Windows)
)
```

**Change to:**
```go
import (
    "database/sql"
    _ "modernc.org/sqlite"  // ← NEW (pure Go, no GCC needed)
)
```

**Find line ~31 (in Connect function):**
```go
db, err := sql.Open("sqlite3", dbPath)  // ← OLD driver name
```

**Change to:**
```go
db, err := sql.Open("sqlite", dbPath)  // ← NEW driver name (no "3")
```

---

### Step 2: Update Dependencies

```bash
cd janus-v3

# Add the pure Go driver
go get modernc.org/sqlite

# Remove old driver (if present)
go mod tidy
```

---

## Complete Fixed File

Here's what `internal/database/sqlite/sqlite.go` should look like:

```go
package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"  // ← Pure Go SQLite driver
)

// SQLite implements the Database interface
type SQLite struct {
	db   *sql.DB
	path string
}

// New creates a new SQLite database connection
func New(path string) (*SQLite, error) {
	return &SQLite{
		path: path,
	}, nil
}

// Connect establishes a connection to the database
func (s *SQLite) Connect() error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", s.path)  // ← "sqlite" not "sqlite3"
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	s.db = db
	return nil
}

// ... rest of file
```

---

## Why This Change?

### Old Driver: `github.com/mattn/go-sqlite3`
- ❌ Requires CGO (C compiler)
- ❌ Needs GCC/MinGW on Windows
- ❌ More complex build setup
- ✅ Slightly faster
- Uses: `"sqlite3"` as driver name

### New Driver: `modernc.org/sqlite`
- ✅ Pure Go (no CGO required)
- ✅ No compiler needed
- ✅ Works on any Windows system
- ✅ Easier to build/distribute
- ⚠️ Slightly slower (negligible for most uses)
- Uses: `"sqlite"` as driver name

---

## Quick Fix Script

```bash
# Navigate to project
cd janus-v3

# Update the import
sed -i 's|github.com/mattn/go-sqlite3|modernc.org/sqlite|g' internal/database/sqlite/sqlite.go

# Update the driver name
sed -i 's|sql.Open("sqlite3"|sql.Open("sqlite"|g' internal/database/sqlite/sqlite.go

# Get the new driver
go get modernc.org/sqlite

# Clean up dependencies
go mod tidy

# Test
go build ./cmd/janus-cli
```

---

## Verify the Fix

### Check the imports:
```bash
grep "import" internal/database/sqlite/sqlite.go -A 5
```

Should show:
```go
import (
    "database/sql"
    // ...
    _ "modernc.org/sqlite"  // ← This line
)
```

### Check the driver name:
```bash
grep 'sql.Open' internal/database/sqlite/sqlite.go
```

Should show:
```go
db, err := sql.Open("sqlite", s.path)  // ← "sqlite" not "sqlite3"
```

---

## Alternative: Use Old Driver with CGO

If you want to keep using `github.com/mattn/go-sqlite3`:

### Install GCC (Windows)
1. Download TDM-GCC: https://jmeubank.github.io/tdm-gcc/
2. Install with default options
3. Build with: `CGO_ENABLED=1 go build`

**But we recommend the pure Go driver instead!**

---

## Common Mistakes

### Mistake 1: Driver name mismatch
```go
import _ "modernc.org/sqlite"
// ...
sql.Open("sqlite3", ...)  // ← WRONG - should be "sqlite"
```

### Mistake 2: Missing underscore import
```go
import "modernc.org/sqlite"  // ← WRONG - needs underscore
```

**Correct:**
```go
import _ "modernc.org/sqlite"  // ← RIGHT - underscore for side-effect import
```

### Mistake 3: Forgot go mod tidy
After changing the import, always run:
```bash
go get modernc.org/sqlite
go mod tidy
```

---

## Test It Works

```bash
# Build CLI
go build ./cmd/janus-cli

# Test database connection
./janus-cli health

# Or test directly
./janus-cli scenario list
```

Should work without errors! ✅

---

## If You Still Get Errors

### Error: "cannot find package modernc.org/sqlite"
```bash
go get modernc.org/sqlite
go mod tidy
```

### Error: "build constraints exclude all Go files"
You're on Windows and have the old CGO driver. Follow the solution above.

### Error: "undefined: sqlite"
Driver name is wrong. Use `"sqlite"` not `"sqlite3"` with modernc.org driver.

---

## Summary

**Two changes needed:**

1. Import: `_ "modernc.org/sqlite"` (not go-sqlite3)
2. Driver: `sql.Open("sqlite", ...)` (not "sqlite3")

Then run:
```bash
go get modernc.org/sqlite
go mod tidy
go build
```

✅ Should work on any platform without requiring GCC!
