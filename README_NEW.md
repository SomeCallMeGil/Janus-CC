# Janus v3 - Enhanced Data Generation System

> Flexible test data generation with PII/Filler distribution control, disk space validation, and cross-platform support.

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)](https://github.com)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

---

## 🚀 Quick Start

```bash
# Clone and build
git clone https://github.com/YOUR_USERNAME/janus-v3.git
cd janus-v3
go mod tidy
go build -o janus-cli ./cmd/janus-cli

# Test it works
go run test_enhanced.go
```

**Enhanced generation is already installed and ready to use!** ✅

---

## ✨ What's Included

This repository contains a complete Janus v3 system with **enhanced generation pre-installed**:

- ✅ **Enhanced Generation Backend** (2,292 lines, 10 modules)
- ✅ **CLI with `gen quick` command**
- ✅ **Platform-specific implementations** (Windows, Linux, Mac)
- ✅ **Pure Go SQLite driver** (no CGO needed)
- ✅ **Comprehensive documentation** (15 guides)

---

## 📦 Installation

```bash
# Install dependencies
go get modernc.org/sqlite
go mod tidy

# Build
go build -o janus-cli ./cmd/janus-cli
go build -o janus-server ./cmd/janus-server
```

That's it! No setup scripts needed - everything is ready to go.

---

## 🎯 Usage Examples

### Generate 100 files with 10% PII
```bash
janus-cli gen quick --file-count 100 --pii-percent 10 --filler-percent 90
```

### Generate 5GB of data
```bash
janus-cli gen quick --total-size 5GB --pii-percent 20 --filler-percent 80
```

### Generate healthcare data
```bash
janus-cli gen quick \
  --file-count 50 \
  --pii-type healthcare \
  --pii-percent 100 \
  --filler-percent 0
```

See [docs/ENHANCED_GENERATION_USAGE.md](docs/ENHANCED_GENERATION_USAGE.md) for more examples.

---

## 📚 Documentation

- **[Quick Start](docs/QUICK_ACTION_CHECKLIST.md)** - Get started in 5 minutes
- **[Complete Guide](docs/PROJECT_STATUS_RECAP.md)** - Full feature overview
- **[Architecture](docs/BACKEND_COMPLETE.md)** - System design
- **[Troubleshooting](docs/COMPILATION_CHECKLIST.md)** - Common issues

[View all docs →](docs/)

---

## 🏗️ Project Structure

```
janus-v3/
├── cmd/
│   ├── janus-cli/       # CLI application
│   └── janus-server/    # Server application
├── internal/
│   ├── api/             # API handlers, websockets
│   ├── core/
│   │   └── generator/   # Enhanced generation ✨
│   │       ├── enhanced/      # Main orchestrator
│   │       ├── models/        # Data structures
│   │       ├── validator/     # Validation + disk checking
│   │       ├── resolver/      # Constraint resolution
│   │       ├── filler/        # Lorem ipsum generation
│   │       └── pii/           # PII data generation
│   └── database/        # SQLite integration
├── docs/                # Complete documentation
└── test_enhanced.go     # Quick test script
```

---

## 🧪 Testing

```bash
# Quick test (10 seconds)
go run test_enhanced.go

# CLI test
./janus-cli gen quick --file-count 10 --pii-percent 50 --filler-percent 50
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file.

---

**Ready to use! Start with `go run test_enhanced.go` to verify everything works.** 🚀
