// Package config provides configuration management for Janus with support
// for both standalone and distributed deployments.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// DeploymentMode defines how Janus is deployed
type DeploymentMode string

const (
	ModeStandalone  DeploymentMode = "standalone"  // Single machine
	ModeDistributed DeploymentMode = "distributed" // Multi-machine with agents
)

// Config represents the complete Janus configuration
type Config struct {
	Mode     DeploymentMode `mapstructure:"mode"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Queue    QueueConfig    `mapstructure:"queue"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Agent    AgentConfig    `mapstructure:"agent"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig configures the API server
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	GRPCPort     int           `mapstructure:"grpc_port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	TLS          TLSConfig     `mapstructure:"tls"`
}

// TLSConfig for secure communication
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig configures the database backend
type DatabaseConfig struct {
	Type         string `mapstructure:"type"` // sqlite, postgres
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Name         string `mapstructure:"name"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	Path         string `mapstructure:"path"` // For SQLite
}

// QueueConfig configures the job queue
type QueueConfig struct {
	Type     string `mapstructure:"type"` // memory, redis
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// StorageConfig configures file storage
type StorageConfig struct {
	Type      string `mapstructure:"type"` // local, s3
	Path      string `mapstructure:"path"` // For local
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// AgentConfig configures agent behavior
type AgentConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	ServerURL        string        `mapstructure:"server_url"`
	ID               string        `mapstructure:"id"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	WorkDir          string        `mapstructure:"work_dir"`
	MaxConcurrent    int           `mapstructure:"max_concurrent"`
}

// LoggingConfig configures logging
type LoggingConfig struct {
	Level      string `mapstructure:"level"` // debug, info, warn, error
	Format     string `mapstructure:"format"` // json, text
	OutputPath string `mapstructure:"output_path"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults for standalone mode
	setDefaults(v)

	// Read config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	// Environment variables override config file
	v.SetEnvPrefix("JANUS")
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Mode
	v.SetDefault("mode", ModeStandalone)

	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.grpc_port", 9090)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.tls.enabled", false)

	// Database (SQLite for standalone)
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.path", "./janus.db")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)

	// Queue (Memory for standalone)
	v.SetDefault("queue.type", "memory")

	// Storage (Local for standalone)
	v.SetDefault("storage.type", "local")
	v.SetDefault("storage.path", "./data")

	// Agent
	v.SetDefault("agent.enabled", false)
	v.SetDefault("agent.heartbeat_interval", "30s")
	v.SetDefault("agent.work_dir", "./payloads")
	v.SetDefault("agent.max_concurrent", 5)

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output_path", "stdout")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate mode
	if c.Mode != ModeStandalone && c.Mode != ModeDistributed {
		return fmt.Errorf("invalid mode: %s (must be standalone or distributed)", c.Mode)
	}

	// Validate database
	if c.Database.Type != "sqlite" && c.Database.Type != "postgres" {
		return fmt.Errorf("invalid database type: %s", c.Database.Type)
	}

	if c.Database.Type == "sqlite" && c.Database.Path == "" {
		return fmt.Errorf("database path required for sqlite")
	}

	if c.Database.Type == "postgres" {
		if c.Database.Host == "" || c.Database.Name == "" {
			return fmt.Errorf("host and name required for postgres")
		}
	}

	// Validate queue
	if c.Queue.Type != "memory" && c.Queue.Type != "redis" {
		return fmt.Errorf("invalid queue type: %s", c.Queue.Type)
	}

	// Validate storage
	if c.Storage.Type != "local" && c.Storage.Type != "s3" {
		return fmt.Errorf("invalid storage type: %s", c.Storage.Type)
	}

	// Distributed mode requires PostgreSQL and Redis
	if c.Mode == ModeDistributed {
		if c.Database.Type != "postgres" {
			return fmt.Errorf("distributed mode requires postgres database")
		}
		if c.Queue.Type != "redis" {
			return fmt.Errorf("distributed mode requires redis queue")
		}
	}

	return nil
}

// IsDistributed returns true if running in distributed mode
func (c *Config) IsDistributed() bool {
	return c.Mode == ModeDistributed
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	switch c.Database.Type {
	case "sqlite":
		return c.Database.Path
	case "postgres":
		sslMode := c.Database.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host, c.Database.Port, c.Database.User,
			c.Database.Password, c.Database.Name, sslMode)
	default:
		return ""
	}
}

// GetRedisAddr returns the Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Queue.Host, c.Queue.Port)
}

// SaveExample saves an example configuration file
func SaveExample(path string) error {
	example := `# Janus Configuration File

# Deployment mode: standalone (single machine) or distributed (multi-machine)
mode: standalone

server:
  host: 0.0.0.0
  port: 8080
  grpc_port: 9090
  read_timeout: 30s
  write_timeout: 30s
  tls:
    enabled: false
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem

# Database configuration
database:
  # Type: sqlite (single node) or postgres (distributed)
  type: sqlite
  path: ./janus.db
  
  # For PostgreSQL (distributed mode)
  # type: postgres
  # host: localhost
  # port: 5432
  # name: janus
  # user: janus
  # password: secure_password
  # ssl_mode: disable
  # max_open_conns: 25
  # max_idle_conns: 5

# Job queue configuration
queue:
  # Type: memory (single node) or redis (distributed)
  type: memory
  
  # For Redis (distributed mode)
  # type: redis
  # host: localhost
  # port: 6379
  # password: ""
  # db: 0

# Storage configuration
storage:
  # Type: local or s3
  type: local
  path: ./data
  
  # For S3
  # type: s3
  # bucket: janus-payloads
  # region: us-east-1
  # access_key: YOUR_ACCESS_KEY
  # secret_key: YOUR_SECRET_KEY

# Agent configuration (for distributed mode)
agent:
  enabled: false
  server_url: localhost:9090
  id: agent-1
  heartbeat_interval: 30s
  work_dir: ./payloads
  max_concurrent: 5

# Logging configuration
logging:
  level: info  # debug, info, warn, error
  format: text # text or json
  output_path: stdout

# Environment Variables:
# All config values can be overridden with environment variables:
# JANUS_MODE=distributed
# JANUS_DATABASE_TYPE=postgres
# JANUS_DATABASE_HOST=localhost
# etc.
`
	return os.WriteFile(path, []byte(example), 0644)
}
