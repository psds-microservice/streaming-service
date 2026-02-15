package model

import "time"

// SessionStatus represents streaming session state.
type SessionStatus string

const (
	SessionStatusWaiting  SessionStatus = "waiting"
	SessionStatusActive   SessionStatus = "active"
	SessionStatusFinished SessionStatus = "finished"
)

// Session is the API view of a streaming session (not GORM entity).
type Session struct {
	ID         string        `json:"id"`
	ClientID   string        `json:"client_id"`
	StreamKey  string        `json:"stream_key"`
	Status     SessionStatus `json:"status"`
	Operators  []Operator    `json:"operators"`
	CreatedAt  time.Time     `json:"created_at"`
	FinishedAt *time.Time    `json:"finished_at,omitempty"`
}

// Operator is a participant (operator) in a session — API response DTO.
type Operator struct {
	UserID      string    `json:"user_id"`
	ConnectedAt time.Time `json:"connected_at"`
}

// CreateSessionRequest is the request body for POST /sessions.
type CreateSessionRequest struct {
	ClientID string `json:"client_id" binding:"required"`
}

// CreateSessionResponse is the response for POST /sessions.
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
	StreamKey string `json:"stream_key"`
	WSURL     string `json:"ws_url"`
	Status    string `json:"status"`
}

// SessionOperatorsResponse is the response for GET /sessions/:id/operators.
type SessionOperatorsResponse struct {
	SessionID string     `json:"session_id"`
	Operators []Operator `json:"operators"`
}
