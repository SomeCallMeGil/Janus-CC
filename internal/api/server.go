// Package api provides the HTTP API server.
package api

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	
	"janus/internal/api/handlers"
	"janus/internal/api/websocket"
	"janus/internal/config"
	"janus/internal/database/models"
)

//go:embed static/*
var staticFiles embed.FS

// Server represents the API server
type Server struct {
	config  *config.Config
	db      models.Database
	router  *chi.Mux
	hub     *websocket.Hub
	server  *http.Server
}

// New creates a new API server
func New(cfg *config.Config, db models.Database) *Server {
	s := &Server{
		config: cfg,
		db:     db,
		router: chi.NewRouter(),
		hub:    websocket.NewHub(),
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// Request ID
	s.router.Use(middleware.RequestID)
	
	// Real IP
	s.router.Use(middleware.RealIP)
	
	// Logger
	s.router.Use(middleware.Logger)
	
	// Recoverer
	s.router.Use(middleware.Recoverer)
	
	// Timeout
	s.router.Use(middleware.Timeout(60 * time.Second))
	
	// CORS — AllowCredentials requires explicit origins, not wildcard
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080", "http://127.0.0.1:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Create handlers
	h := handlers.New(s.db, s.hub)

	// API routes
	s.router.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", h.Health)

		// Scenarios
		r.Route("/scenarios", func(r chi.Router) {
			r.Get("/", h.ListScenarios)
			r.Post("/", h.CreateScenario)
			r.Get("/templates", h.ListTemplates)
			
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetScenario)
				r.Put("/", h.UpdateScenario)
				r.Delete("/", h.DeleteScenario)
				r.Get("/stats", h.GetScenarioStats)
				
				// Actions
				r.Post("/generate", h.GenerateFiles)
				r.Post("/encrypt", h.EncryptFiles)
				r.Post("/destroy", h.DestroyScenario)
				
				// Files
				r.Get("/files", h.ListFiles)
				r.Get("/manifest", h.GetManifest)
				r.Get("/export", h.ExportCSV)
			})
		})

		// Jobs
		r.Route("/jobs", func(r chi.Router) {
			r.Get("/", h.ListJobs)
			r.Post("/", h.CreateJob)
			
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.GetJob)
				r.Delete("/", h.CancelJob)
				r.Get("/progress", h.GetJobProgress)
			})
		})

		// Enhanced generation
		r.Post("/generate/enhanced", h.EnhancedGenerate)

		// Activity
		r.Get("/activity", h.GetActivity)
	})

	// WebSocket endpoint
	s.router.Get("/ws/v1/activity", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(s.hub, w, r)
	})

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err == nil {
		s.router.Handle("/*", http.FileServer(http.FS(staticFS)))
	} else {
		// Fallback to serving from disk if embed fails
		s.router.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir("./web/static"))))
	}
}

// Start starts the API server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	// Start WebSocket hub
	go s.hub.Run()

	log.Printf("Starting server on %s", addr)
	log.Printf("Web UI: http://localhost:%d", s.config.Server.Port)
	log.Printf("API: http://localhost:%d/api/v1", s.config.Server.Port)
	log.Printf("WebSocket: ws://localhost:%d/ws/v1/activity", s.config.Server.Port)

	if s.config.Server.TLS.Enabled {
		return s.server.ListenAndServeTLS(
			s.config.Server.TLS.CertFile,
			s.config.Server.TLS.KeyFile,
		)
	}

	return s.server.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down server...")
	
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	
	return nil
}

// Router returns the router (for testing)
func (s *Server) Router() *chi.Mux {
	return s.router
}
