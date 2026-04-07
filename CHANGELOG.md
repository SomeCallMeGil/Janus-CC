# Changelog

All notable changes to Janus are documented in this file.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [3.1.1] - 2026-04-07

### Fixed
- **Encryption error suppression** — `EncryptJob` always returned `nil` regardless of outcome. Jobs where every file failed now return a proper error and the `encryption_failed` WebSocket event fires correctly.
- **Silent per-file failures** — the progress callback broadcast `file_encrypted` for every file regardless of result. Failed files now broadcast `file_failed` with the specific error message.
- **Windows file lock on rename** — `defer src.Close()` left the source file open when `os.Rename` attempted to overwrite it. Windows holds an exclusive lock on open files; fixed by closing the source explicitly before the rename. All early-exit error paths also close the source.
- **Encryption no-op after prior failure** — a previously failed run marked all files `failed` in the database. The scheduler only queried `pending` files, found zero candidates, and silently completed with no work done. Failed files are now included as retry candidates on subsequent runs.
- **No visual indicator on encrypted files** — encrypted files were modified in place with no rename. Files are now renamed to append the `.janus` extension (e.g. `report.csv` → `report.csv.janus`) as a visual indicator. The new path is persisted to the database.

---

## [3.1.0] - 2026-04-06

### Added
- **Profile system** — named, reusable generation configurations stored in SQLite
  - `profile create` — define a profile with all generation parameters
  - `profile list` — tabular view of all saved profiles
  - `profile show <id>` — full profile detail
  - `profile update <id>` — partial field updates without overwriting the profile
  - `profile delete <id>` — remove a profile with confirmation prompt
  - `gen profile <id>` — generate data from a saved profile; supports `--output` override and `--watch` streaming
- **Six REST API endpoints** for profile management (`/api/v1/profiles`)
- **Five built-in profiles** seeded on first run: `quick-pii-test`, `mixed-realistic`, `healthcare-large`, `financial-audit`, `compliance-10pct`
- **Profile-based generation** with per-run option overrides that do not mutate the stored profile
- `internal/version` package — single source of truth for the release version string
- **Comprehensive documentation**
  - `docs/PROFILE_SYSTEM.md` — concept guide, use cases, migration from flags
  - `docs/API_REFERENCE.md` — all endpoints with curl examples and request/response shapes
  - `docs/CLI_REFERENCE.md` — all CLI commands with flag tables and examples
  - Updated `README.md` with profile quick-start section

### Changed
- Generation job lifecycle (pause/resume/cancel) wired through the job registry for consistent state management
- Performance and database improvements to the enhanced generation pipeline
- Technical debt removed from handler layer

### Fixed
- UI pause/resume/cancel flow for active generation jobs
- Server 404 routing edge case on unknown paths
- SQLite driver compatibility on all target platforms

---

## [3.0.0] - Initial Release

- Enhanced generation backend with PII/filler distribution control
- `gen quick` CLI command
- Disk space validation before generation starts
- Cross-platform support (Windows, Linux, macOS)
- Pure Go SQLite driver (no CGO)
- WebSocket progress streaming
- Scenario management API
