# CLI Reference

The `janus-cli` binary communicates with a running `janus-server` instance. Set the server address with the `--api` global flag (default: `http://localhost:8080`).

```
janus-cli [command] [flags]
```

---

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--api` | `http://localhost:8080` | Server base URL |

---

## profile

Manage named, reusable generation configurations.

```
janus-cli profile <subcommand>
```

---

### profile create

Create and save a new profile.

```
janus-cli profile create [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | (required) | Profile name (must be unique) |
| `--description` | `""` | Human-readable notes |
| `--file-count` | `0` | Number of files (count mode) |
| `--total-size` | `""` | Total dataset size, e.g. `5GB` (size mode) |
| `--file-size-min` | `1KB` | Minimum individual file size |
| `--file-size-max` | `10MB` | Maximum individual file size |
| `--pii-percent` | `10` | PII content fraction 0–100 |
| `--pii-type` | `standard` | PII category: `standard`, `healthcare`, `financial` |
| `--filler-percent` | `90` | Filler content fraction 0–100 |
| `--output` | `./payloads` | Output directory |
| `--seed` | `0` | Reproducibility seed (0 = random) |

Set either `--file-count` (count mode) or `--total-size` (size mode). `pii-percent` and `filler-percent` must sum to 100.

**Examples**

```bash
# Healthcare — count mode
janus-cli profile create \
  --name "Healthcare Test" \
  --file-count 1000 \
  --pii-type healthcare \
  --pii-percent 100 \
  --filler-percent 0 \
  --output ./payloads/healthcare

# Mixed enterprise — size mode
janus-cli profile create \
  --name "Enterprise Mixed" \
  --total-size 5GB \
  --pii-percent 15 \
  --filler-percent 85 \
  --output ./payloads/enterprise

# Reproducible test — fixed seed
janus-cli profile create \
  --name "Regression Baseline" \
  --file-count 500 \
  --pii-percent 20 \
  --filler-percent 80 \
  --seed 42
```

**Output**
```
Created profile: Healthcare Test (ID: 550e8400-e29b-41d4-a716-446655440000)
```

---

### profile list

List all saved profiles.

```
janus-cli profile list
```

**Output**
```
ID                                    NAME                DESCRIPTION
550e8400-e29b-41d4-a716-446655440000  Healthcare Test     HIPAA validation dataset
a1b2c3d4-e5f6-7890-abcd-ef1234567890  Enterprise Mixed    Mixed enterprise load
```

---

### profile show

Show full details of a profile.

```
janus-cli profile show <id>
```

**Arguments**

| Argument | Description |
|----------|-------------|
| `id` | Profile UUID (from `profile list`) |

**Example**
```bash
janus-cli profile show 550e8400-e29b-41d4-a716-446655440000
```

**Output** — full profile JSON including all options, timestamps, and ID.

---

### profile update

Apply partial updates to an existing profile. Only the flags you pass are changed; everything else stays the same.

```
janus-cli profile update <id> [flags]
```

**Arguments**

| Argument | Description |
|----------|-------------|
| `id` | Profile UUID |

**Flags**

| Flag | Description |
|------|-------------|
| `--name` | New profile name |
| `--description` | New description |
| `--file-count` | New file count |
| `--total-size` | New total size |
| `--file-size-min` | New minimum file size |
| `--file-size-max` | New maximum file size |
| `--pii-percent` | New PII percent |
| `--filler-percent` | New filler percent |
| `--pii-type` | New PII type |
| `--output` | New output directory |

**Examples**
```bash
# Adjust PII distribution
janus-cli profile update 550e8400-... --pii-percent 25 --filler-percent 75

# Rename
janus-cli profile update 550e8400-... --name "Healthcare Test v2"
```

**Output**
```
Updated profile: Healthcare Test v2
```

---

### profile delete

Delete a profile. Prompts for confirmation before proceeding.

```
janus-cli profile delete <id>
```

**Arguments**

| Argument | Description |
|----------|-------------|
| `id` | Profile UUID |

**Example**
```bash
janus-cli profile delete 550e8400-e29b-41d4-a716-446655440000
# Delete profile "Healthcare Test"? This cannot be undone. [y/N]: y
# Deleted profile: 550e8400-e29b-41d4-a716-446655440000
```

---

## gen

Generate data. Supports two modes: one-off generation (`gen quick`) and profile-based generation (`gen profile`).

---

### gen quick

Generate data using inline flags (no profile).

```
janus-cli gen quick [flags]
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--file-count` | `0` | Number of files (count mode) |
| `--total-size` | `""` | Total size (size mode) |
| `--file-size-min` | `1KB` | Minimum file size |
| `--file-size-max` | `10MB` | Maximum file size |
| `--pii-percent` | `10` | PII percent 0–100 |
| `--pii-type` | `standard` | `standard`, `healthcare`, `financial` |
| `--filler-percent` | `90` | Filler percent 0–100 |
| `--output` | `./payloads` | Output directory |
| `--seed` | `0` | Reproducibility seed |
| `--watch` | `false` | Stream progress until complete |

**Examples**
```bash
# 100 files, 10% PII
janus-cli gen quick --file-count 100 --pii-percent 10 --filler-percent 90

# 5GB size-mode
janus-cli gen quick --total-size 5GB --pii-percent 20 --filler-percent 80

# Healthcare, all PII
janus-cli gen quick \
  --file-count 50 \
  --pii-type healthcare \
  --pii-percent 100 \
  --filler-percent 0
```

---

### gen profile

Generate data using a saved profile. The profile's stored options are used; `--output` overrides the stored path for this run only without modifying the profile.

```
janus-cli gen profile <id> [flags]
```

**Arguments**

| Argument | Description |
|----------|-------------|
| `id` | Profile UUID |

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `""` | Override output directory (this run only) |
| `--watch` | `false` | Stream progress via WebSocket until complete |

**Examples**
```bash
# Run with stored settings
janus-cli gen profile 550e8400-e29b-41d4-a716-446655440000

# Override output directory
janus-cli gen profile 550e8400-e29b-41d4-a716-446655440000 --output /tmp/test

# Stream progress
janus-cli gen profile 550e8400-e29b-41d4-a716-446655440000 --watch
```

**Output (without --watch)**
```
Generation started — scenario ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**Output (with --watch)** — live progress lines until the job completes.

---

## encrypt

Encrypt a percentage of files in a scenario. Runs in the background on the server; the command returns immediately after the job is accepted.

Encrypted files are renamed with a `.janus` extension (e.g. `report.csv` → `report.csv.janus`). If a previous encryption attempt failed, those files are automatically retried.

```
janus-cli encrypt <scenario-id> [flags]
```

**Arguments**

| Argument | Description |
|----------|-------------|
| `id` | Scenario UUID (from `scenario list`) |

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--password`, `-w` | (required) | Encryption password |
| `--percentage`, `-p` | `25.0` | Percentage of files to encrypt (0–100) |
| `--mode`, `-m` | `partial` | `partial` — encrypt first 4096 bytes only; `full` — encrypt entire file |

**Examples**
```bash
# Encrypt 25% of files with partial mode
janus-cli encrypt a1b2c3d4-e5f6-7890-abcd-ef1234567890 --password secret

# Encrypt all files using full mode
janus-cli encrypt a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  --password secret \
  --percentage 100 \
  --mode full
```

**Output**
```
✓ Encryption started for scenario: a1b2c3d4-e5f6-7890-abcd-ef1234567890
  Percentage: 100.0%
  Mode: full
  (This runs in the background. Check server logs for progress)
```

---

## Other Commands

### health

Check server health.

```
janus-cli health
```

### scenario

Manage existing generation scenarios.

```
janus-cli scenario list
janus-cli scenario show <id>
```

### job

Control active generation jobs.

```
janus-cli job pause <scenario-id>
janus-cli job resume <scenario-id>
janus-cli job cancel <scenario-id>
```
