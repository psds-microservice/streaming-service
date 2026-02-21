package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds streaming-service configuration (shape as user-service template).
type Config struct {
	AppEnv   string // APP_ENV
	AppHost  string // APP_HOST
	HTTPPort string // APP_PORT or HTTP_PORT
	LogLevel string // LOG_LEVEL

	// PostgreSQL (nested as in template)
	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		Database string
		SSLMode  string
	}

	// WebSocket
	WSReadBufferSize  int
	WSWriteBufferSize int
	WSMaxMessageSize  int64

	// Session
	SessionMaxOperators int
	SessionIdleTimeout  int // seconds

	// WebSocket URL returned in CreateSession (e.g. wss://stream.example.com)
	WSBaseURL string

	// Recording: copy stream to recording-service, then set URL in session-manager
	EnableRecording        bool   // ENABLE_RECORDING
	RecordingServiceAddr   string // RECORDING_SERVICE_ADDR (gRPC, e.g. localhost:8096)
	SessionManagerGRPCAddr string // SESSION_MANAGER_GRPC_ADDR (e.g. localhost:8091)
}

// parseIntEnv parses key from env; on error uses default and returns the default value.
func parseIntEnv(envKey, defaultVal string) (int, error) {
	s := getEnv(envKey, defaultVal)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("config %s: invalid integer %q: %w", envKey, s, err)
	}
	return n, nil
}

// parseInt64Env parses key from env as int64; on error returns error.
func parseInt64Env(envKey, defaultVal string) (int64, error) {
	s := getEnv(envKey, defaultVal)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("config %s: invalid integer %q: %w", envKey, s, err)
	}
	return n, nil
}

// Load loads config from environment (.env if present).
func Load() (*Config, error) {
	_ = godotenv.Load()

	readBuf, err := parseIntEnv("WS_READ_BUFFER_SIZE", "4096")
	if err != nil {
		return nil, err
	}
	writeBuf, err := parseIntEnv("WS_WRITE_BUFFER_SIZE", "4096")
	if err != nil {
		return nil, err
	}
	maxMsg, err := parseInt64Env("WS_MAX_MESSAGE_SIZE", "10485760")
	if err != nil {
		return nil, err
	}
	maxOps, err := parseIntEnv("SESSION_MAX_OPERATORS", "10")
	if err != nil {
		return nil, err
	}
	idleTO, err := parseIntEnv("SESSION_IDLE_TIMEOUT", "3600")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		AppHost:             getEnv("APP_HOST", "0.0.0.0"),
		HTTPPort:            firstEnv("APP_PORT", "HTTP_PORT", "8090"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		WSReadBufferSize:    readBuf,
		WSWriteBufferSize:   writeBuf,
		WSMaxMessageSize:    maxMsg,
		SessionMaxOperators: maxOps,
		SessionIdleTimeout:  idleTO,
		WSBaseURL:           getEnv("WS_BASE_URL", ""),
	}
	cfg.DB.Host = getEnv("DB_HOST", "localhost")
	cfg.DB.Port = getEnv("DB_PORT", "5432")
	cfg.DB.User = getEnv("DB_USER", "postgres")
	cfg.DB.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.DB.Database = getEnv("DB_DATABASE", "streaming_service")
	cfg.DB.SSLMode = getEnv("DB_SSLMODE", "disable")
	cfg.EnableRecording = getEnv("ENABLE_RECORDING", "false") == "true" || getEnv("ENABLE_RECORDING", "false") == "1"
	cfg.RecordingServiceAddr = getEnv("RECORDING_SERVICE_ADDR", "localhost:8096")
	cfg.SessionManagerGRPCAddr = getEnv("SESSION_MANAGER_GRPC_ADDR", "localhost:9091")
	return cfg, nil
}

// Validate checks required fields and production safety.
func (c *Config) Validate() error {
	if c.DB.Host == "" {
		return errors.New("config: DB_HOST is required")
	}
	if c.DB.User == "" {
		return errors.New("config: DB_USER is required")
	}
	if c.DB.Database == "" {
		return errors.New("config: DB_DATABASE is required")
	}
	if c.AppEnv == "production" && c.DB.Password == "" {
		return errors.New("config: in production DB_PASSWORD is required")
	}
	return nil
}

// DSN returns PostgreSQL connection string for GORM.
func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Database, c.DB.SSLMode)
}

// DatabaseURL returns postgres URL for golang-migrate (postgres://user:pass@host:port/dbname?sslmode=...).
func (c *Config) DatabaseURL() string {
	pass := url.QueryEscape(c.DB.Password)
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DB.User, pass, c.DB.Host, c.DB.Port, c.DB.Database, c.DB.SSLMode)
}

// Addr returns listen address for HTTP server.
func (c *Config) Addr() string {
	return c.AppHost + ":" + c.HTTPPort
}

func firstEnv(keysAndDef ...string) string {
	if len(keysAndDef) == 0 {
		return ""
	}
	def := keysAndDef[len(keysAndDef)-1]
	keys := keysAndDef[:len(keysAndDef)-1]
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return def
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
