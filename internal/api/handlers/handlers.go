// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"janus/internal/api/websocket"
	"janus/internal/core/encryptor"
	"janus/internal/core/generator"
	"janus/internal/core/generator/enhanced"
	"janus/internal/core/scenario"
	"janus/internal/core/scheduler"
	"janus/internal/core/tracker"
	"janus/internal/database/models"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	db              models.Database
	hub             *websocket.Hub
	scenarioManager *scenario.Manager
	generator       *generator.Generator
	scheduler       *scheduler.Scheduler
	tracker         *tracker.Tracker
}

// New creates new handlers
func New(db models.Database, hub *websocket.Hub) *Handlers {
	return &Handlers{
		db:              db,
		hub:             hub,
		scenarioManager: scenario.New(db),
		generator:       generator.New(db),
		scheduler:       scheduler.New(db),
		tracker:         tracker.New(db),
	}
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Health checks server health
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(); err != nil {
		respondError(w, http.StatusServiceUnavailable, "Database unavailable")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

// ListScenarios lists all scenarios
func (h *Handlers) ListScenarios(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.scenarioManager.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

// CreateScenario creates a new scenario
func (h *Handlers) CreateScenario(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Template    string                   `json:"template"`
		Config      *scenario.ScenarioConfig `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var s *models.Scenario
	var err error

	if req.Template != "" {
		// Create from template
		s, err = h.scenarioManager.CreateFromTemplate(req.Name, req.Template)
	} else if req.Config != nil {
		// Create from config
		createReq := scenario.CreateRequest{
			Name:        req.Name,
			Description: req.Description,
			Config:      *req.Config,
		}
		s, err = h.scenarioManager.Create(createReq)
	} else {
		respondError(w, http.StatusBadRequest, "Either template or config required")
		return
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast event
	h.hub.Broadcast(map[string]interface{}{
		"type": "scenario_created",
		"id":   s.ID,
		"name": s.Name,
		"time": time.Now().UTC(),
	})

	respondJSON(w, http.StatusCreated, s)
}

// GetScenario gets a scenario by ID
func (h *Handlers) GetScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	s, err := h.scenarioManager.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Scenario not found")
		return
	}

	respondJSON(w, http.StatusOK, s)
}

// UpdateScenario updates a scenario
func (h *Handlers) UpdateScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s, err := h.scenarioManager.Update(id, updates)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, s)
}

// DeleteScenario deletes a scenario
func (h *Handlers) DeleteScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.scenarioManager.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.hub.Broadcast(map[string]interface{}{
		"type": "scenario_deleted",
		"id":   id,
		"time": time.Now().UTC(),
	})

	respondJSON(w, http.StatusOK, map[string]string{"message": "Scenario deleted"})
}

// ListTemplates lists available templates
func (h *Handlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.scenarioManager.ListTemplates()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
	})
}

// GetScenarioStats gets scenario statistics
func (h *Handlers) GetScenarioStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	stats, err := h.scenarioManager.GetStats(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GenerateFiles generates files for a scenario
func (h *Handlers) GenerateFiles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Get scenario config
	config, err := h.scenarioManager.GetConfig(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Start generation in background
	go func() {
		h.hub.Broadcast(map[string]interface{}{
			"type":        "generation_started",
			"scenario_id": id,
			"time":        time.Now().UTC(),
		})

		opts := generator.GenerateOptions{
			ScenarioID: id,
			Config:     config,
			Progress: func(current, total int, message string) {
				h.hub.Broadcast(map[string]interface{}{
					"type":        "generation_progress",
					"scenario_id": id,
					"current":     current,
					"total":       total,
					"message":     message,
					"time":        time.Now().UTC(),
				})
			},
		}

		if err := h.generator.Generate(opts); err != nil {
			h.hub.Broadcast(map[string]interface{}{
				"type":        "generation_failed",
				"scenario_id": id,
				"error":       err.Error(),
				"time":        time.Now().UTC(),
			})
			return
		}

		h.hub.Broadcast(map[string]interface{}{
			"type":        "generation_completed",
			"scenario_id": id,
			"time":        time.Now().UTC(),
		})
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Generation started",
	})
}

// EncryptFiles encrypts files in a scenario
func (h *Handlers) EncryptFiles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Percentage float64 `json:"percentage"`
		Password   string  `json:"password"`
		Mode       string  `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Password == "" {
		respondError(w, http.StatusBadRequest, "Password required")
		return
	}

	// Start encryption in background
	go func() {
		enc := encryptor.New(h.db, req.Password, 4)

		opts := encryptor.DefaultOptions()
		if req.Mode == "full" {
			opts.Mode = encryptor.ModeFullEncryption
		}

		h.hub.Broadcast(map[string]interface{}{
			"type":        "encryption_started",
			"scenario_id": id,
			"percentage":  req.Percentage,
			"time":        time.Now().UTC(),
		})

		err := enc.EncryptScenario(id, req.Percentage, opts, func(result encryptor.EncryptResult) {
			h.hub.Broadcast(map[string]interface{}{
				"type":        "file_encrypted",
				"scenario_id": id,
				"file_id":     result.FileID,
				"file_path":   result.FilePath,
				"time":        time.Now().UTC(),
			})
		})

		if err != nil {
			h.hub.Broadcast(map[string]interface{}{
				"type":        "encryption_failed",
				"scenario_id": id,
				"error":       err.Error(),
				"time":        time.Now().UTC(),
			})
			return
		}

		h.hub.Broadcast(map[string]interface{}{
			"type":        "encryption_completed",
			"scenario_id": id,
			"time":        time.Now().UTC(),
		})
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Encryption started",
	})
}

// DestroyScenario deletes all generated payload files for a scenario from disk
func (h *Handlers) DestroyScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	files, err := h.db.ListFilesByScenario(id, models.FileFilters{})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	go func() {
		h.hub.Broadcast(map[string]interface{}{
			"type":        "destroy_started",
			"scenario_id": id,
			"total":       len(files),
			"time":        time.Now().UTC(),
		})

		deleted := 0
		failed := 0
		for _, f := range files {
			if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) {
				failed++
				continue
			}
			deleted++
		}

		// Mark scenario status as destroyed
		scenario, err := h.scenarioManager.Get(id)
		if err == nil {
			h.scenarioManager.Update(id, map[string]interface{}{
				"status": "destroyed",
			})
			_ = scenario
		}

		h.hub.Broadcast(map[string]interface{}{
			"type":        "destroy_completed",
			"scenario_id": id,
			"deleted":     deleted,
			"failed":      failed,
			"time":        time.Now().UTC(),
		})
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"scenario_id": id,
		"message":     "Destroy started",
		"total_files": len(files),
	})
}

// ListFiles lists files in a scenario
func (h *Handlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Parse query parameters
	status := r.URL.Query().Get("status")
	dataType := r.URL.Query().Get("data_type")
	limitStr := r.URL.Query().Get("limit")

	filters := models.FileFilters{
		Status:   models.FileStatus(status),
		DataType: dataType,
	}

	if limitStr != "" {
		limit, _ := strconv.Atoi(limitStr)
		filters.Limit = limit
	}

	files, err := h.db.ListFilesByScenario(id, filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"files": files,
		"count": len(files),
	})
}

// GetManifest gets the file manifest
func (h *Handlers) GetManifest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	manifest, err := h.tracker.GenerateManifest(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, manifest)
}

// ExportCSV exports manifest to CSV
func (h *Handlers) ExportCSV(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Create temp file
	tmpFile := fmt.Sprintf("/tmp/manifest-%s-%d.csv", id, time.Now().Unix())

	if err := h.tracker.ExportCSV(id, tmpFile); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Set headers for download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=manifest-%s.csv", id))

	http.ServeFile(w, r, tmpFile)
}

// ListJobs lists jobs
func (h *Handlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	scenarioID := r.URL.Query().Get("scenario_id")
	statusStr := r.URL.Query().Get("status")

	filters := models.JobFilters{
		ScenarioID: scenarioID,
		Status:     models.JobStatus(statusStr),
	}

	jobs, err := h.db.ListJobs(filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// CreateJob creates a new job
func (h *Handlers) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID       string    `json:"scenario_id"`
		ScheduledAt      time.Time `json:"scheduled_at"`
		TargetPercentage float64   `json:"target_percentage"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	job, err := h.scheduler.ScheduleJob(req.ScenarioID, req.ScheduledAt, req.TargetPercentage)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, job)
}

// GetJob gets a job by ID
func (h *Handlers) GetJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	job, err := h.scheduler.GetJob(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	respondJSON(w, http.StatusOK, job)
}

// CancelJob cancels a job
func (h *Handlers) CancelJob(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if err := h.scheduler.CancelJob(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Job cancelled"})
}

// GetJobProgress gets job progress
func (h *Handlers) GetJobProgress(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	progress, err := h.scheduler.GetJobProgress(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, progress)
}

// EnhancedGenerate runs enhanced generation from a QuickGenerateOptions payload
func (h *Handlers) EnhancedGenerate(w http.ResponseWriter, r *http.Request) {
	var req enhanced.QuickGenerateOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	go func() {
		h.hub.Broadcast(map[string]interface{}{
			"type": "enhanced_generation_started",
			"name": req.Name,
			"time": time.Now().UTC(),
		})

		result, err := enhanced.QuickGenerate(h.db, req, func(p enhanced.Progress) {
			h.hub.Broadcast(map[string]interface{}{
				"type":           "enhanced_generation_progress",
				"name":           req.Name,
				"current":        p.Current,
				"total":          p.Total,
				"percent":        p.Percent,
				"current_file":   p.CurrentFile,
				"bytes_written":  p.BytesWritten,
				"status":         p.Status,
				"time":           time.Now().UTC(),
			})
		})

		if err != nil {
			h.hub.Broadcast(map[string]interface{}{
				"type":  "enhanced_generation_failed",
				"name":  req.Name,
				"error": err.Error(),
				"time":  time.Now().UTC(),
			})
			return
		}

		h.hub.Broadcast(map[string]interface{}{
			"type":          "enhanced_generation_completed",
			"name":          req.Name,
			"files_created": result.FilesCreated,
			"bytes_written": result.BytesWritten,
			"duration_ms":   result.Duration.Milliseconds(),
			"time":          time.Now().UTC(),
		})
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "Enhanced generation started",
	})
}

// GetActivity gets activity log
func (h *Handlers) GetActivity(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	filters := models.ActivityFilters{
		Limit: limit,
	}

	logs, err := h.db.GetActivityLogs(filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}
