# Profile System Guide

Profiles are named, reusable generation configurations stored in the Janus database. Instead of repeating long flag lists, you define a profile once and invoke it by ID any time you need that dataset.

---

## Concepts

### What is a profile?

A profile bundles every parameter needed for a generation run:

| Field | Description |
|-------|-------------|
| `name` | Human-readable identifier (unique) |
| `description` | Optional notes |
| `output_path` | Default output directory |
| `total_size` | Size-mode constraint (e.g. `"5GB"`) |
| `file_count` | Count-mode constraint (e.g. `1000`) |
| `file_size_min` | Minimum individual file size (e.g. `"1KB"`) |
| `file_size_max` | Maximum individual file size (e.g. `"10MB"`) |
| `pii_percent` | Fraction of content that is PII (0–100) |
| `pii_type` | PII category: `standard`, `healthcare`, `financial` |
| `filler_percent` | Fraction of content that is filler (0–100) |
| `formats` | File formats to produce: `csv`, `json`, `txt` |
| `directory_depth` | Subdirectory nesting level (default: 3) |
| `seed` | Integer seed for reproducible output (0 = random) |
| `workers` | Parallel file writers (0 = CPU count) |

**Size mode vs count mode:** Set either `total_size` or `file_count`, not both. `total_size` takes precedence when both are present.

**PII + filler must total 100.** Validation enforces this before any files are written.

---

## Built-in Profiles

Five profiles are seeded automatically when the server starts for the first time. They demonstrate the most common testing patterns and are ready to run immediately.

| Name | Mode | Size/Count | PII % | PII Type | Description |
|------|------|-----------|-------|----------|-------------|
| `quick-pii-test` | count | 1,000 files | 100% | standard | Fast smoke test |
| `mixed-realistic` | size | 1 GB | 15% | standard | Typical enterprise dataset |
| `healthcare-large` | size | 5 GB | 30% | healthcare | Medical records simulation |
| `financial-audit` | count | 5,000 files | 40% | financial | Financial records simulation |
| `compliance-10pct` | size | 10 GB | 10% | standard | Compliance testing at scale |

Run any of them immediately:

```bash
# List to find the ID
janus-cli profile list

# Run it
janus-cli gen profile <id>
```

---

## Use Cases

### Regression testing

Create a profile with a fixed seed so every run produces an identical dataset:

```bash
janus-cli profile create \
  --name "Regression Baseline" \
  --file-count 500 \
  --pii-percent 20 \
  --filler-percent 80 \
  --seed 42 \
  --output ./payloads/regression
```

Re-run it after any code change to compare output byte-for-byte.

### Load testing

A large size-mode profile for sustained load:

```bash
janus-cli profile create \
  --name "Load Test 50GB" \
  --total-size 50GB \
  --pii-percent 10 \
  --filler-percent 90 \
  --output ./payloads/load
```

### Compliance scanning

Simulate a realistic enterprise filesystem with low PII density:

```bash
janus-cli profile create \
  --name "Enterprise Scan" \
  --total-size 10GB \
  --pii-type standard \
  --pii-percent 10 \
  --filler-percent 90 \
  --file-size-min 1MB \
  --file-size-max 100MB \
  --output ./payloads/compliance
```

### Healthcare / HIPAA testing

100% healthcare PII for scanner validation:

```bash
janus-cli profile create \
  --name "HIPAA Validation" \
  --file-count 1000 \
  --pii-type healthcare \
  --pii-percent 100 \
  --filler-percent 0 \
  --output ./payloads/hipaa
```

---

## Common Workflows

### Create → run → iterate

```bash
# 1. Create
janus-cli profile create --name "My Profile" --file-count 100 \
  --pii-percent 50 --filler-percent 50

# 2. Find the ID
janus-cli profile list

# 3. Run
janus-cli gen profile <id>

# 4. Adjust
janus-cli profile update <id> --pii-percent 75 --filler-percent 25

# 5. Run again
janus-cli gen profile <id>
```

### Override at run time

The `--output` flag on `gen profile` overrides the stored path without changing the profile:

```bash
janus-cli gen profile <id> --output /tmp/scratch
```

### Watch progress

Pass `--watch` to stream live progress until the job finishes:

```bash
janus-cli gen profile <id> --watch
```

---

## Migration from Flags

If you currently run `gen quick` with a fixed set of flags, convert it to a profile:

**Before:**
```bash
janus-cli gen quick \
  --file-count 2000 \
  --pii-type financial \
  --pii-percent 40 \
  --filler-percent 60 \
  --file-size-min 5KB \
  --file-size-max 5MB \
  --output ./payloads/financial
```

**After:**
```bash
# One-time setup
janus-cli profile create \
  --name "Financial Dataset" \
  --file-count 2000 \
  --pii-type financial \
  --pii-percent 40 \
  --filler-percent 60 \
  --file-size-min 5KB \
  --file-size-max 5MB \
  --output ./payloads/financial

# Every subsequent run
janus-cli gen profile <id>
```

---

## API Access

All profile operations are also available via REST. See [API_REFERENCE.md](API_REFERENCE.md) for full endpoint documentation.

Quick example:

```bash
# List profiles
curl http://localhost:8080/api/v1/profiles

# Generate from profile
curl -X POST http://localhost:8080/api/v1/profiles/<id>/generate
```
