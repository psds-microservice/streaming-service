package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/psds-microservice/streaming-service/internal/errs"
	"github.com/psds-microservice/streaming-service/internal/model"
	"github.com/psds-microservice/streaming-service/internal/service"
)

// SessionHandler handles REST API for sessions.
type SessionHandler struct {
	svc service.SessionServicer
	cfg *service.WSConfig
}

// WSConfig exposes base URL for WebSocket (e.g. for response ws_url).
type WSConfig struct {
	BaseURL string
}

// NewSessionHandler creates a session handler (D: принимает SessionServicer).
func NewSessionHandler(svc service.SessionServicer, wsBaseURL string) *SessionHandler {
	return &SessionHandler{
		svc: svc,
		cfg: &service.WSConfig{BaseURL: wsBaseURL},
	}
}

// CreateSession godoc
// POST /sessions
func (h *SessionHandler) CreateSession(c *gin.Context) {
	var req model.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "message": err.Error()})
		return
	}
	sess, err := h.svc.Create(req.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	wsURL := h.cfg.WSURL(sess.ID, req.ClientID)
	c.JSON(http.StatusCreated, model.CreateSessionResponse{
		SessionID: sess.ID,
		StreamKey: sess.StreamKey,
		WSURL:     wsURL,
		Status:    string(sess.Status),
	})
}

// DeleteSession godoc
// DELETE /sessions/:id
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}
	err := h.svc.Finish(sessionID)
	if err != nil {
		if errors.Is(err, errs.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finish session"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetSessionOperators godoc
// GET /sessions/:id/operators
func (h *SessionHandler) GetSessionOperators(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}
	operators, err := h.svc.GetOperators(sessionID)
	if err != nil {
		if errors.Is(err, errs.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get operators"})
		return
	}
	c.JSON(http.StatusOK, model.SessionOperatorsResponse{
		SessionID: sessionID,
		Operators: operators,
	})
}
