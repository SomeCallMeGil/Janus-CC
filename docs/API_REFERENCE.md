# API Reference

Base URL: `http://localhost:8080`

All endpoints accept and return `application/json`. Error responses use the shape `{"error": "message"}`.

---

## Health

### GET /health

Check server and database availability.

**Response 200**
```json
{
  "status": "ok",
  "time": "2026-04-06T12:00:00Z"
}
```

**Response 503** — database unavailable.

---

## Profiles

Profiles are named, reusable generation configurations. All six endpoints live under `/api/v1/profiles`.

---

### GET /api/v1/profiles

List all saved profiles.

**Response 200**
```json
{
  "count": 2,
  "profiles": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Healthcare Test",
      "description": "HIPAA validation dataset",
      "options": {
        "output_path": "./payloads/healthcare",
        "file_count": 1000,
        "file_size_min": "1KB",
        "file_size_max": "10MB",
        "pii_percent": 100,
        "pii_type": "healthcare",
        "filler_percent": 0,
        "formats": ["csv", "json"],
        "directory_depth": 3,
        "seed": 0,
        "workers": 0
      },
      "created_at": "2026-04-06T10:00:00Z",
      "updated_at": "2026-04-06T10:00:00Z"
    }
  ]
}
```

**curl**
```bash
curl http://localhost:8080/api/v1/profiles
```

---

### POST /api/v1/profiles

Create a new profile.

**Request body**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Unique profile name |
| `description` | string | no | Human-readable notes |
| `options` | object | yes | Generation options (see below) |

**Options object**

| Field | Type | Description |
|-------|------|-------------|
| `output_path` | string | Output directory |
| `total_size` | string | Size mode, e.g. `"5GB"` (overrides `file_count`) |
| `file_count` | int | Count mode, e.g. `1000` |
| `file_size_min` | string | Minimum file size, e.g. `"1KB"` |
| `file_size_max` | string | Maximum file size, e.g. `"10MB"` |
| `pii_percent` | float | PII fraction 0–100 |
| `pii_type` | string | `standard`, `healthcare`, or `financial` |
| `filler_percent` | float | Filler fraction 0–100 (must sum to 100 with `pii_percent`) |
| `formats` | string[] | File formats: `csv`, `json`, `txt` |
| `directory_depth` | int | Subdirectory depth (default: 3) |
| `seed` | int64 | Reproducibility seed (0 = random) |
| `workers` | int | Parallel writers (0 = CPU count) |

**Example**
```bash
curl -X POST http://localhost:8080/api/v1/profiles \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Healthcare Test",
    "description": "HIPAA validation dataset",
    "options": {
      "output_path": "./payloads/healthcare",
      "file_count": 1000,
      "pii_type": "healthcare",
      "pii_percent": 100,
      "filler_percent": 0
    }
  }'
```

**Response 201** — returns the created profile object (same shape as list items above).

**Response 400** — `name` missing, name already exists, or validation error.

---

### GET /api/v1/profiles/{id}

Retrieve a single profile by ID.

**Path parameter**

| Parameter | Description |
|-----------|-------------|
| `id` | Profile UUID |

**Response 200** — profile object.

**Response 404** — profile not found.

**curl**
```bash
curl http://localhost:8080/api/v1/profiles/550e8400-e29b-41d4-a716-446655440000
```

---

### PUT /api/v1/profiles/{id}

Apply a partial update to a profile. Send only the fields you want to change. Nested `options` fields are merged — you do not need to resend the entire options object.

**Request body** — any subset of the create body fields.

**Example: change PII percent only**
```bash
curl -X PUT http://localhost:8080/api/v1/profiles/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{"options": {"pii_percent": 25, "filler_percent": 75}}'
```

**Example: rename and update description**
```bash
curl -X PUT http://localhost:8080/api/v1/profiles/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{"name": "Healthcare Test v2", "description": "Updated for Q2 audit"}'
```

**Response 200** — updated profile object.

**Response 400** — validation error or name collision.

**Response 404** — profile not found.

---

### DELETE /api/v1/profiles/{id}

Delete a profile by ID.

**curl**
```bash
curl -X DELETE http://localhost:8080/api/v1/profiles/550e8400-e29b-41d4-a716-446655440000
```

**Response 200**
```json
{"message": "Profile deleted"}
```

**Response 500** — deletion failed.

---

### POST /api/v1/profiles/{id}/generate

Start a generation job using the saved profile. The profile's stored options are used as-is; optional overrides in the request body are merged on top without modifying the saved profile.

**Request body** (optional) — any `options` fields to override for this run only.

**Example: run with stored settings**
```bash
curl -X POST http://localhost:8080/api/v1/profiles/550e8400-e29b-41d4-a716-446655440000/generate
```

**Example: override output path**
```bash
curl -X POST http://localhost:8080/api/v1/profiles/550e8400-e29b-41d4-a716-446655440000/generate \
  -H "Content-Type: application/json" \
  -d '{"output_path": "/tmp/test-run"}'
```

**Response 202**
```json
{
  "message": "Generation started",
  "scenario_id": "a1b2c3d4-...",
  "profile_id": "550e8400-..."
}
```

Use `scenario_id` to track progress via the WebSocket at `ws://localhost:8080/ws` or to pause/resume/cancel via the scenario control endpoints.

**Response 404** — profile not found.

**Response 500** — generation failed to start.

---

## Scenarios

Scenarios are the underlying generation records created when generation starts (either via `gen quick` or from a profile). Use the returned `scenario_id` to monitor or control the run.

### GET /api/v1/scenarios

List all scenarios.

### GET /api/v1/scenarios/{id}

Get scenario details and status.

### POST /api/v1/scenarios/{id}/pause

Pause an active generation job.

### POST /api/v1/scenarios/{id}/resume

Resume a paused generation job.

### POST /api/v1/scenarios/{id}/cancel

Cancel an active or paused generation job.

---

## WebSocket

### WS /ws

Connect to receive real-time generation events.

**Event types**

| Type | Description |
|------|-------------|
| `generation_started` | Job started; includes `scenario_id` and optionally `profile_id` |
| `generation_progress` | Progress update with file counts and bytes written |
| `generation_complete` | Job finished successfully |
| `generation_failed` | Job encountered a fatal error |
| `generation_paused` | Job paused |
| `generation_resumed` | Job resumed |
| `generation_cancelled` | Job cancelled |

**Example progress event**
```json
{
  "type": "generation_progress",
  "scenario_id": "a1b2c3d4-...",
  "files_created": 250,
  "files_total": 1000,
  "bytes_written": 52428800,
  "elapsed_seconds": 12
}
```
