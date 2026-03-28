# Handlers.go Compilation Fixes

## Issues Fixed

### Issue 1: Unused "id" variable (Line 329) ✅
**Error:** `id declared and not used`
**Location:** `internal/api/handlers/handlers.go` line 329
**Function:** `DestroyScenario()`

**Problem:**
```go
func (h *Handlers) DestroyScenario(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")  // ← Declared but never used
    
    // TODO: Implement destruction logic
    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Destruction not yet implemented",
    })
}
```

**Fix:**
```go
func (h *Handlers) DestroyScenario(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    
    // TODO: Implement destruction logic
    respondJSON(w, http.StatusOK, map[string]string{
        "scenario_id": id,  // ← Now using the id variable
        "message":     "Destruction not yet implemented",
    })
}
```

---

### Issue 2: WriteStatus vs WriteHeader (Line 47)
**Error:** `WriteStatus undefined`
**Location:** `internal/api/handlers/handlers.go` line 47
**Function:** `respondJSON()`

**If you have this error, your file has:**
```go
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteStatus(status)  // ← WRONG - this method doesn't exist
    json.NewEncoder(w).Encode(data)
}
```

**Should be:**
```go
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)  // ← CORRECT - WriteHeader (not WriteStatus)
    json.NewEncoder(w).Encode(data)
}
```

**Note:** The handlers-fixed.go file already has this correct. If you're seeing this error, you may have an older version of the file.

---

## How to Apply

### Option 1: Use the Fixed File
```bash
cp handlers-fixed.go internal/api/handlers/handlers.go
```

### Option 2: Manual Fix

**Fix 1 - Line 329 (DestroyScenario):**
Add `"scenario_id": id,` to the response map

**Fix 2 - Line 47 (respondJSON) - IF YOU HAVE THIS ERROR:**
Change `w.WriteStatus(status)` to `w.WriteHeader(status)`

---

## Verify the Fixes

### Check Line 329
```bash
sed -n '327,335p' internal/api/handlers/handlers.go
```

**Should show:**
```go
// DestroyScenario destroys a scenario's payload
func (h *Handlers) DestroyScenario(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    
    // TODO: Implement destruction logic
    respondJSON(w, http.StatusOK, map[string]string{
        "scenario_id": id,      // ← id is used here
        "message":     "Destruction not yet implemented",
    })
}
```

### Check Line 47
```bash
sed -n '44,49p' internal/api/handlers/handlers.go
```

**Should show:**
```go
// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)    // ← WriteHeader (not WriteStatus)
    json.NewEncoder(w).Encode(data)
}
```

---

## Test Compilation

```bash
go build ./internal/api/handlers/
```

Should compile without errors! ✅

---

## Complete File Location

**Original:** `internal/api/handlers/handlers.go` (in your janus-v3 project)
**Fixed:** `handlers-fixed.go` (download from outputs)

---

## Why These Errors Occur

### Unused Variable Error
Go is strict about unused variables. If you declare `id := chi.URLParam(r, "id")` but never use `id`, the compiler will fail. This is a design decision to prevent dead code and catch potential bugs.

**Solution:** Either use the variable or remove it (or prefix with `_` if intentionally unused).

### WriteStatus Error  
`http.ResponseWriter` has a method called `WriteHeader()` not `WriteStatus()`. This is likely a typo or autocomplete error.

**Correct methods:**
- ✅ `WriteHeader(int)` - sets the status code
- ❌ `WriteStatus(int)` - doesn't exist

---

## Summary

| Line | Error | Fix |
|------|-------|-----|
| 329 | `id declared and not used` | Use id in response map |
| 47 | `WriteStatus undefined` | Change to WriteHeader |

Both are now fixed in `handlers-fixed.go`! ✅
