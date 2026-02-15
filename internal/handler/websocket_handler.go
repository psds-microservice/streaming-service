package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/psds-microservice/streaming-service/internal/model"
	"github.com/psds-microservice/streaming-service/internal/service"
	"go.uber.org/zap"
)

// StreamWSHandler handles WebSocket connections for /ws/stream/:session_id/:user_id.
type StreamWSHandler struct {
	hub    *service.StreamHub
	sess   *service.SessionService
	logger *zap.Logger
}

// NewStreamWSHandler creates the WebSocket stream handler.
func NewStreamWSHandler(hub *service.StreamHub, sess *service.SessionService, logger *zap.Logger) *StreamWSHandler {
	return &StreamWSHandler{hub: hub, sess: sess, logger: logger}
}

// ServeWS upgrades the request to WebSocket and runs the stream loop.
// Path: /ws/stream/:session_id/:user_id
// First connection with user_id == session.ClientID is the stream source (client); others are operators.
func (h *StreamWSHandler) ServeWS(c *gin.Context) {
	sessionID := c.Param("session_id")
	userID := c.Param("user_id")
	if sessionID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and user_id required"})
		return
	}

	sess, err := h.sess.Get(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	if sess.Status == model.SessionStatusFinished {
		c.JSON(http.StatusGone, gin.H{"error": "session already finished"})
		return
	}

	conn, err := h.hub.Upgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	role := service.PeerRoleOperator
	if userID == sess.ClientID {
		role = service.PeerRoleClient
	}

	peer, cleanup := h.hub.Register(sessionID, userID, role, conn)
	defer cleanup()

	if role == service.PeerRoleOperator {
		if err := h.sess.AddOperator(sessionID, userID); err != nil {
			h.logger.Warn("failed to add operator to session", zap.Error(err))
			return
		}
	}

	// Writer goroutine: send from peer.Send to connection
	go h.writePump(peer)

	// Reader: receive from client and relay to operators
	h.readPump(peer)
}

func (h *StreamWSHandler) readPump(p *service.Peer) {
	defer func() {
		_ = p.Conn.Close()
	}()
	for {
		mt, data, err := p.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Debug("read error", zap.Error(err))
			}
			break
		}
		if p.Role == service.PeerRoleClient {
			h.hub.RelayToOperators(p.SessionID, mt, data)
		}
		// Operators don't send media back in this minimal version; could add later
	}
}

func (h *StreamWSHandler) writePump(p *service.Peer) {
	defer func() {
		_ = p.Conn.Close()
	}()
	for data := range p.Send {
		if err := p.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			break
		}
	}
}
