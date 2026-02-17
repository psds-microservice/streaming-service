package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health and ready checks.
type HealthHandler struct{}

// NewHealthHandler creates a health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health responds to GET /health.
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "streaming-service",
		"time":    time.Now().Unix(),
	})
}

// Ready responds to GET /ready (for k8s readiness). Формат {"status": "ready"} для единообразия с остальными сервисами.
func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
