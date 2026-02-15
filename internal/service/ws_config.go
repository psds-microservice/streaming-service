package service

import "fmt"

// WSConfig holds WebSocket URL base for responses.
type WSConfig struct {
	BaseURL string
}

// WSURL returns the WebSocket URL for a session and user (e.g. wss://host/ws/stream/sessionID/userID).
func (c *WSConfig) WSURL(sessionID, userID string) string {
	if c == nil || c.BaseURL == "" {
		return fmt.Sprintf("/ws/stream/%s/%s", sessionID, userID)
	}
	base := c.BaseURL
	if base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	return fmt.Sprintf("%s/ws/stream/%s/%s", base, sessionID, userID)
}
