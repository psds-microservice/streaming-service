package application

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/psds-microservice/streaming-service/internal/config"
	"github.com/psds-microservice/streaming-service/internal/database"
	"github.com/psds-microservice/streaming-service/internal/handler"
	"github.com/psds-microservice/streaming-service/internal/router"
	"github.com/psds-microservice/streaming-service/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// API is the HTTP + WebSocket API application.
type API struct {
	cfg *config.Config
	srv *http.Server
	db  *gorm.DB
}

// NewAPI creates the API application: validates config, runs migrations, opens DB, builds router.
func NewAPI(cfg *config.Config) (*API, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := database.MigrateUp(cfg.DatabaseURL()); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	db, err := database.Open(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	logger, _ := zap.NewProduction()
	if cfg.AppEnv == "development" {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	hub := service.NewStreamHub(cfg.WSMaxMessageSize, logger)
	hub.SetReadLimit(cfg.WSMaxMessageSize)
	sessionSvc := service.NewSessionService(db, cfg, hub)
	sessionHandler := handler.NewSessionHandler(sessionSvc, cfg.WSBaseURL)
	streamWS := handler.NewStreamWSHandler(hub, sessionSvc, logger)
	health := handler.NewHealthHandler()

	r := router.New(sessionHandler, streamWS, health)

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &API{cfg: cfg, srv: srv, db: db}, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled; then shuts down gracefully.
func (a *API) Run(ctx context.Context) error {
	addr := a.srv.Addr
	host := a.cfg.AppHost
	if host == "0.0.0.0" {
		host = "localhost"
	}
	base := "http://" + host + ":" + a.cfg.HTTPPort
	log.Printf("streaming-service HTTP listening on %s", addr)
	log.Printf("  Health:   %s/health", base)
	log.Printf("  Ready:    %s/ready", base)
	log.Printf("  Sessions: %s/sessions", base)
	log.Printf("  WebSocket: %s/ws/stream/:session_id/:user_id", base)

	go func() {
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
