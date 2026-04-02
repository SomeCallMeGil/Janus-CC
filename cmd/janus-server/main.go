// Janus Server - API server for security testing platform
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
			log.Fatal().Err(err).Msg("load config")
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

	// Configure zerolog from loaded config
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil || level == zerolog.NoLevel {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	if cfg.Logging.Format != "json" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Str("mode", string(cfg.Mode)).Msg("starting Janus Server v3.0")

	// Initialize database
	db, err := sqlite.New(cfg.Database.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("create database")
	}

	if err := db.Connect(); err != nil {
		log.Fatal().Err(err).Msg("connect to database")
	}
	defer db.Close()

	log.Info().Str("path", cfg.Database.Path).Msg("database connected")

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("migrate database")
	}

	log.Info().Msg("database migrations completed")

	// Create and start API server
	server := api.New(cfg, db)

	// Handle graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Info().Msg("received shutdown signal")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}

		os.Exit(0)
	}()

	// Start server
	log.Info().Msg("server starting")
	if err := server.Start(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
