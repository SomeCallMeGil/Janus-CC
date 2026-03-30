// Janus Server - API server for security testing platform
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"janus/internal/api"
	"janus/internal/config"
	"janus/internal/database/sqlite"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to configuration file")
	dbPath := flag.String("db", "./janus.db", "Path to database file")
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()

	// Load configuration
	var cfg *config.Config
	var err error
	
	if *configPath != "" {
		cfg, err = config.Load(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		// Use defaults
		cfg = &config.Config{
			Mode: config.ModeStandalone,
			Server: config.ServerConfig{
				Host:         "0.0.0.0",
				Port:         *port,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			Database: config.DatabaseConfig{
				Type: "sqlite",
				Path: *dbPath,
			},
			Logging: config.LoggingConfig{
				Level:  "info",
				Format: "text",
			},
		}
	}

	log.Printf("Starting Janus Server v3.0")
	log.Printf("Mode: %s", cfg.Mode)

	// Initialize database
	db, err := sqlite.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("Database: %s", cfg.Database.Path)

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database migrations completed")

	// Create and start API server
	server := api.New(cfg, db)

	// Handle graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Received shutdown signal")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		os.Exit(0)
	}()

	// Start server
	log.Println("Server starting...")
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
