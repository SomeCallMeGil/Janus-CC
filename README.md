# Janus v3 - Enhanced Data Generation System

> Flexible test data generation with PII/Filler distribution control, disk space validation, and cross-platform support.

[![Version](https://img.shields.io/badge/version-3.1.0-blue.svg)](CHANGELOG.md)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)](https://github.com)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

---

## Quick Start

```bash
# Clone and build
git clone https://github.com/YOUR_USERNAME/janus-v3.git
cd janus-v3
go mod tidy
go build -o janus-cli ./cmd/janus-cli
go build -o janus-server ./cmd/janus-server

# Start the server
./janus-server

# Test it works
go run test_enhanced.go
```

---

## What's Included

- Enhanced generation backend with PII/filler distribution control
- CLI with `gen quick` and `gen profile` commands
- Profile system for reusable generation configurations
- Platform-specific implementations (Windows, Linux, macOS)
- Pure Go SQLite driver (no CGO needed)
- REST API with WebSocket progress streaming

---

## Usage Examples

### One-off generation

```bash
# Generate 100 files with 10% PII
janus-cli gen quick --file-count 100 --pii-percent 10 --filler-percent 90

# Generate 5GB of data
janus-cli gen quick --total-size 5GB --pii-percent 20 --filler-percent 80

# Generate healthcare data
janus-cli gen quick \
  --file-count 50 \
  --pii-type healthcare \
  --pii-percent 100 \
  --filler-percent 0
```

---

## Profile System

Profiles are named, reusable generation configurations stored in the database. Create a profile once and run it any time — no flags to remember.

### Create a profile

```bash
# Healthcare test dataset
janus-cli profile create \
  --name "Healthcare Test" \
  --file-count 1000 \
  --pii-type healthcare \
  --pii-percent 100 \
  --filler-percent 0 \
  --output ./payloads/healthcare

# Mixed enterprise dataset
janus-cli profile create \
  --name "Enterprise Mixed" \
  --total-size 5GB \
  --pii-percent 15 \
  --filler-percent 85 \
  --output ./payloads/enterprise
```

### Run a profile

```bash
# Generate using the profile (by ID)
janus-cli gen profile <profile-id>

# Override the output directory at run time
janus-cli gen profile <profile-id> --output /tmp/test-run

# Stream progress until complete
janus-cli gen profile <profile-id> --watch
```

### Manage profiles

```bash
janus-cli profile list              # list all profiles
janus-cli profile show <id>         # show full profile details
janus-cli profile update <id> --pii-percent 25   # update a field
janus-cli profile delete <id>       # delete a profile
```

### Built-in profiles

Five profiles are seeded automatically on first run:

| Name | Mode | Description |
|------|------|-------------|
| `quick-pii-test` | count | 1000 standard PII files — fast smoke test |
| `mixed-realistic` | size | 1 GB mixed dataset — 15% PII, 85% filler |
| `healthcare-large` | size | 5 GB healthcare dataset — 30% medical PII |
| `financial-audit` | count | 5000 financial records — 40% financial PII |
| `compliance-10pct` | size | 10 GB compliance test — 10% PII |

See [docs/PROFILE_SYSTEM.md](docs/PROFILE_SYSTEM.md) for the full profile guide.

---

## Documentation

- [Profile System Guide](docs/PROFILE_SYSTEM.md) — profiles, use cases, migration from flags
- [API Reference](docs/API_REFERENCE.md) — all REST endpoints with curl examples
- [CLI Reference](docs/CLI_REFERENCE.md) — all CLI commands and flags

---

## Project Structure

```
janus-v3/
├── cmd/
│   ├── janus-cli/       # CLI application
│   └── janus-server/    # Server application
├── internal/
│   ├── api/             # API handlers, WebSocket hub
│   ├── core/
│   │   ├── generator/   # Enhanced generation engine
│   │   ├── profiles/    # Profile store and manager
│   │   └── ...
│   └── database/        # SQLite integration
├── docs/                # Documentation
└── test_enhanced.go     # Quick smoke test
```

---

## License

MIT License — see [LICENSE](LICENSE).
