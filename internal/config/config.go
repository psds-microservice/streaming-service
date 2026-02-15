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
}

// Load loads config from environment (.env if present).
func Load() (*Config, error) {
	_ = godotenv.Load()

	readBuf, _ := strconv.Atoi(getEnv("WS_READ_BUFFER_SIZE", "4096"))
	writeBuf, _ := strconv.Atoi(getEnv("WS_WRITE_BUFFER_SIZE", "4096"))
	maxMsg, _ := strconv.ParseInt(getEnv("WS_MAX_MESSAGE_SIZE", "10485760"), 10, 64)
	maxOps, _ := strconv.Atoi(getEnv("SESSION_MAX_OPERATORS", "10"))
	idleTO, _ := strconv.Atoi(getEnv("SESSION_IDLE_TIMEOUT", "3600"))

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
