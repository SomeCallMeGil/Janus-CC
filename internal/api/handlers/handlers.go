// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"janus/internal/api/jobs"
	"janus/internal/api/websocket"
	"janus/internal/core/encryptor"
	"janus/internal/core/generator"
	"janus/internal/core/generator/enhanced"
	"janus/internal/core/profiles"
	"janus/internal/core/scenario"
	"janus/internal/core/scheduler"
	"janus/internal/core/tracker"
	"janus/internal/database/models"
	"janus/internal/database/sqlite"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	db              models.Database
	hub             *websocket.Hub
	scenarioManager *scenario.Manager
	generator       *generator.Generator
	scheduler       *scheduler.Scheduler
	tracker         *tracker.Tracker
	profileManager  *profiles.Manager
	jobRegistry     *jobs.Registry
}

// New creates new handlers
func New(db models.Database, hub *websocket.Hub) *Handlers {
	var profileMgr *profiles.Manager
	if sqlDB, ok := db.(*sqlite.SQLiteDB); ok {
		store := profiles.NewSQLiteStore(sqlDB)
		profileMgr = profiles.New(store)
		seedDefaultProfiles(profileMgr)
	}

	return &Handlers{
		db:              db,
		hub:             hub,
		scenarioManager: scenario.New(db),
		generator:       generator.New(db),
		scheduler:       scheduler.New(db),
		tracker:         tracker.New(db),
		profileManager:  profileMgr,
		jobRegistry:     jobs.NewRegistry(),
	}
}

// seedDefaultProfiles inserts built-in profiles if they don't already exist.
func seedDefaultProfiles(mgr *profiles.Manager) {
	for _, d := range profiles.DefaultProfiles {
		opts := d.Options
		opts.Name = d.Name
		mgr.Create(d.Name, d.Description, opts) // silently skips if name already exists
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
		Workers    int     `json:"workers"`
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
		workers := req.Workers
		if workers < 1 {
			workers = 4
		}
		enc := encryptor.New(h.db, req.Password, workers)

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

		// Remove the payload directory tree entirely (leaves no empty dirs behind).
		// For enhanced scenarios created before this fix (Config="{}"), fall back to
		// deriving the root from the common ancestor of the tracked file paths.
		root := ""
		if cfg, err := h.scenarioManager.GetConfig(id); err == nil {
			root = cfg.Generation.Root
		}
		if root == "" && len(files) > 0 {
			// Walk up from the first file until we find a directory that is an
			// ancestor of all tracked files — use the parent of the deepest common prefix.
			root = filepath.Dir(files[0].Path)
			for _, f := range files[1:] {
				for !strings.HasPrefix(f.Path, root+string(os.PathSeparator)) && root != filepath.Dir(root) {
					root = filepath.Dir(root)
				}
			}
		}
		if root != "" {
			if err := os.RemoveAll(root); err != nil {
				log.Error().Err(err).Str("scenario_id", id).Str("root", root).Msg("destroy scenario: remove tree")
				h.hub.Broadcast(map[string]interface{}{
					"type":        "destroy_warning",
					"scenario_id": id,
					"message":     fmt.Sprintf("directory removal failed: %v", err),
					"time":        time.Now().UTC(),
				})
			}
		}

		// Mark scenario status as destroyed
		h.scenarioManager.Update(id, map[string]interface{}{
			"status": "destroyed",
		})

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
		if n, err := strconv.Atoi(limitStr); err == nil {
			filters.Limit = n
		}
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

// ExportCSV streams the file manifest as CSV directly to the response — no temp file.
func (h *Handlers) ExportCSV(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=manifest-%s.csv", id))

	if err := h.tracker.WriteCSV(id, w); err != nil {
		// Headers already sent — can't send a JSON error, log it instead.
		log.Error().Err(err).Str("scenario_id", id).Msg("export CSV")
	}
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
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

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
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	if err := h.scheduler.CancelJob(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Job cancelled"})
}

// GetJobProgress gets job progress
func (h *Handlers) GetJobProgress(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	progress, err := h.scheduler.GetJobProgress(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, progress)
}

// runEnhancedGeneration creates a scenario record, registers job control, and starts the
// generation goroutine. profileID is empty for direct (non-profile) generation.
// Returns the scenario ID so the caller can include it in the HTTP response.
func (h *Handlers) runEnhancedGeneration(opts enhanced.QuickGenerateOptions, descPrefix, profileID string) (string, error) {
	scenarioRecord := &models.Scenario{
		ID:          uuid.New().String(),
		Name:        opts.Name,
		Description: fmt.Sprintf("%s: %s", descPrefix, opts.OutputPath),
		Type:        "enhanced",
		Config:      fmt.Sprintf(`{"generation":{"root":%q}}`, opts.OutputPath),
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.db.CreateScenario(scenarioRecord); err != nil {
		return "", fmt.Errorf("create scenario: %w", err)
	}

	scenarioID := scenarioRecord.ID
	opts.ScenarioID = scenarioID

	ctrl := jobs.NewControl(context.Background())
	h.jobRegistry.Register(scenarioID, ctrl)

	go func() {
		defer h.jobRegistry.Remove(scenarioID)

		startEvent := map[string]interface{}{
			"type":        "enhanced_generation_started",
			"scenario_id": scenarioID,
			"name":        opts.Name,
			"workers":     opts.Workers,
			"time":        time.Now().UTC(),
		}
		if profileID != "" {
			startEvent["profile_id"] = profileID
		}
		h.scenarioManager.UpdateStatus(scenarioID, "generating")
		h.hub.Broadcast(startEvent)

		result, err := enhanced.QuickGenerate(ctrl.Context(), h.db, opts, func(p enhanced.Progress) {
			h.hub.Broadcast(map[string]interface{}{
				"type":          "enhanced_generation_progress",
				"scenario_id":   scenarioID,
				"name":          opts.Name,
				"current":       p.Current,
				"total":         p.Total,
				"percent":       p.Percent,
				"current_file":  p.CurrentFile,
				"bytes_written": p.BytesWritten,
				"status":        p.Status,
				"message":       p.Message,
				"time":          time.Now().UTC(),
			})
		}, ctrl.CheckPoint)

		if err != nil {
			status := "failed"
			if err == context.Canceled {
				status = "cancelled"
			}
			h.scenarioManager.UpdateStatus(scenarioID, status)
			h.hub.Broadcast(map[string]interface{}{
				"type":        "enhanced_generation_failed",
				"scenario_id": scenarioID,
				"name":        opts.Name,
				"error":       err.Error(),
				"cancelled":   err == context.Canceled,
				"time":        time.Now().UTC(),
			})
			return
		}

		h.scenarioManager.UpdateStatus(scenarioID, "ready")
		h.hub.Broadcast(map[string]interface{}{
			"type":          "enhanced_generation_completed",
			"scenario_id":   scenarioID,
			"name":          opts.Name,
			"files_created": result.FilesCreated,
			"bytes_written": result.BytesWritten,
			"duration_ms":   result.Duration.Milliseconds(),
			"time":          time.Now().UTC(),
		})
	}()

	return scenarioID, nil
}

// EnhancedGenerate runs enhanced generation from a QuickGenerateOptions payload.
func (h *Handlers) EnhancedGenerate(w http.ResponseWriter, r *http.Request) {
	var req enhanced.QuickGenerateOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	scenarioID, err := h.runEnhancedGeneration(req, "Enhanced generation", "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":     "Enhanced generation started",
		"scenario_id": scenarioID,
	})
}

// GetActivity gets activity log
func (h *Handlers) GetActivity(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
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

// --- Profile handlers ---

// ListProfiles returns all saved profiles
func (h *Handlers) ListProfiles(w http.ResponseWriter, r *http.Request) {
	if h.profileManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Profile manager unavailable")
		return
	}
	list, err := h.profileManager.List()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"profiles": list,
		"count":    len(list),
	})
}

// CreateProfile creates a new profile
func (h *Handlers) CreateProfile(w http.ResponseWriter, r *http.Request) {
	if h.profileManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Profile manager unavailable")
		return
	}
	var req struct {
		Name        string                       `json:"name"`
		Description string                       `json:"description"`
		Options     enhanced.QuickGenerateOptions `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	p, err := h.profileManager.Create(req.Name, req.Description, req.Options)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, p)
}

// GetProfile retrieves a profile by ID
func (h *Handlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	if h.profileManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Profile manager unavailable")
		return
	}
	id := chi.URLParam(r, "id")
	p, err := h.profileManager.Get(id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, p)
}

// UpdateProfile applies partial updates to a profile
func (h *Handlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if h.profileManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Profile manager unavailable")
		return
	}
	id := chi.URLParam(r, "id")
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	p, err := h.profileManager.Update(id, updates)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, p)
}

// DeleteProfile removes a profile
func (h *Handlers) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	if h.profileManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Profile manager unavailable")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.profileManager.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Profile deleted"})
}

// --- Generation job control handlers ---

// PauseGeneration pauses the active generation job for a scenario.
func (h *Handlers) PauseGeneration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctrl, ok := h.jobRegistry.Get(id)
	if !ok {
		respondError(w, http.StatusNotFound, "no active generation for this scenario")
		return
	}
	ctrl.Pause()
	h.hub.Broadcast(map[string]interface{}{
		"type":        "generation_paused",
		"scenario_id": id,
		"time":        time.Now().UTC(),
	})
	respondJSON(w, http.StatusOK, map[string]string{"message": "paused"})
}

// ResumeGeneration resumes a paused generation job.
func (h *Handlers) ResumeGeneration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctrl, ok := h.jobRegistry.Get(id)
	if !ok {
		respondError(w, http.StatusNotFound, "no active generation for this scenario")
		return
	}
	ctrl.Resume()
	h.hub.Broadcast(map[string]interface{}{
		"type":        "generation_resumed",
		"scenario_id": id,
		"time":        time.Now().UTC(),
	})
	respondJSON(w, http.StatusOK, map[string]string{"message": "resumed"})
}

// CancelGeneration cancels an active or paused generation job.
func (h *Handlers) CancelGeneration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctrl, ok := h.jobRegistry.Get(id)
	if !ok {
		respondError(w, http.StatusNotFound, "no active generation for this scenario")
		return
	}
	ctrl.Cancel()
	h.hub.Broadcast(map[string]interface{}{
		"type":        "generation_cancelled",
		"scenario_id": id,
		"time":        time.Now().UTC(),
	})
	respondJSON(w, http.StatusOK, map[string]string{"message": "cancelled"})
}

// GenerateFromProfile triggers enhanced generation using a saved profile
func (h *Handlers) GenerateFromProfile(w http.ResponseWriter, r *http.Request) {
	if h.profileManager == nil {
		respondError(w, http.StatusServiceUnavailable, "Profile manager unavailable")
		return
	}
	id := chi.URLParam(r, "id")

	// Optional overrides in request body
	var overrides map[string]interface{}
	json.NewDecoder(r.Body).Decode(&overrides) // ignore decode error — overrides are optional

	opts, err := h.profileManager.Resolve(id, overrides)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	scenarioID, err := h.runEnhancedGeneration(opts, "Profile generation", id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":     "Generation started",
		"scenario_id": scenarioID,
		"profile_id":  id,
	})
}
