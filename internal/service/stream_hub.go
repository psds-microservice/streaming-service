package service

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// PeerRole is client (stream source) or operator (stream viewer).
type PeerRole string

const (
	PeerRoleClient   PeerRole = "client"
	PeerRoleOperator PeerRole = "operator"
)

// Peer represents a WebSocket connection in a session.
type Peer struct {
	SessionID string
	UserID    string
	Role      PeerRole
	Conn      *websocket.Conn
	Send      chan []byte
}

// StreamRecorder receives a copy of the client stream for recording (optional).
type StreamRecorder interface {
	WriteChunk(ctx context.Context, sessionID string, data []byte)
	EndSession(ctx context.Context, sessionID string)
}

// StreamHubForHandler — интерфейс для WebSocket handler (D: зависимость от абстракции).
type StreamHubForHandler interface {
	Register(sessionID, userID string, role PeerRole, conn *websocket.Conn) (*Peer, func())
	Upgrader() *websocket.Upgrader
	RelayToOperators(sessionID string, messageType int, data []byte)
}

// StreamHub manages WebSocket connections and relays media per session.
type StreamHub struct {
	mu         sync.RWMutex
	peers      map[string]map[*Peer]struct{} // sessionID -> set of peers
	upgrader   websocket.Upgrader
	maxMsgSize int64
	log        *zap.Logger
	recorder   StreamRecorder  // optional: copy of client stream to recording-service
	ctx        context.Context // app context for recording (shutdown propagation)
}

// SetRecorder sets the optional recorder for copying client stream to recording-service.
func (h *StreamHub) SetRecorder(r StreamRecorder) { h.recorder = r }

// SetContext sets the app context for recording (for shutdown propagation).
func (h *StreamHub) SetContext(ctx context.Context) { h.ctx = ctx }

// NewStreamHub creates a new stream hub.
func NewStreamHub(maxMessageSize int64, log *zap.Logger) *StreamHub {
	return &StreamHub{
		peers:      make(map[string]map[*Peer]struct{}),
		maxMsgSize: maxMessageSize,
		log:        log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024 * 4,
			WriteBufferSize: 1024 * 4,
			// Allow all origins for dev; in prod set CheckOrigin.
		},
	}
}

// SetReadLimit sets max message size for connections.
func (h *StreamHub) SetReadLimit(n int64) { h.maxMsgSize = n }

// Register adds a peer to a session and returns a cleanup function.
func (h *StreamHub) Register(sessionID, userID string, role PeerRole, conn *websocket.Conn) (*Peer, func()) {
	if h.maxMsgSize > 0 {
		conn.SetReadLimit(h.maxMsgSize)
	}
	p := &Peer{
		SessionID: sessionID,
		UserID:    userID,
		Role:      role,
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}
	h.mu.Lock()
	if h.peers[sessionID] == nil {
		h.peers[sessionID] = make(map[*Peer]struct{})
	}
	h.peers[sessionID][p] = struct{}{}
	h.mu.Unlock()

	h.log.Info("peer registered",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("role", string(role)))

	cleanup := func() {
		h.unregister(sessionID, p)
	}
	return p, cleanup
}

func (h *StreamHub) unregister(sessionID string, p *Peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.peers[sessionID]; ok {
		delete(m, p)
		if len(m) == 0 {
			delete(h.peers, sessionID)
		}
	}
	close(p.Send)
	h.log.Info("peer unregistered",
		zap.String("session_id", sessionID),
		zap.String("user_id", p.UserID))
}

// RelayToOperators sends data from the client to all operators in the session.
func (h *StreamHub) RelayToOperators(sessionID string, messageType int, data []byte) {
	h.mu.RLock()
	m, ok := h.peers[sessionID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	// Copy peers so we don't hold lock while writing
	peers := make([]*Peer, 0, len(m))
	for p := range m {
		if p.Role == PeerRoleOperator {
			peers = append(peers, p)
		}
	}
	h.mu.RUnlock()

	for _, p := range peers {
		select {
		case p.Send <- data:
		default:
			h.log.Warn("operator send buffer full", zap.String("user_id", p.UserID))
		}
	}
	if h.recorder != nil && len(data) > 0 {
		ctx := h.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		h.recorder.WriteChunk(ctx, sessionID, data)
	}
}

// CloseSession closes all connections in the session and removes them.
func (h *StreamHub) CloseSession(sessionID string) {
	h.mu.Lock()
	m, ok := h.peers[sessionID]
	if !ok {
		h.mu.Unlock()
		return
	}
	delete(h.peers, sessionID)
	h.mu.Unlock()

	if h.recorder != nil {
		ctx := h.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		h.recorder.EndSession(ctx, sessionID)
	}
	// Send close message then close connections
	closeMsg := map[string]string{"event": "session_finished", "session_id": sessionID}
	raw, _ := json.Marshal(closeMsg)
	for p := range m {
		_ = p.Conn.WriteMessage(websocket.TextMessage, raw)
		close(p.Send)
		_ = p.Conn.Close()
	}
	h.log.Info("session closed", zap.String("session_id", sessionID))
}

// Upgrader returns the WebSocket upgrader for HTTP handlers.
func (h *StreamHub) Upgrader() *websocket.Upgrader {
	return &h.upgrader
}

// PeerCount returns number of peers in a session (for debugging).
func (h *StreamHub) PeerCount(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.peers[sessionID])
}
