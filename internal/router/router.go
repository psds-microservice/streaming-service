package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/psds-microservice/streaming-service/internal/handler"
	"github.com/psds-microservice/streaming-service/pkg/constants"
)

// New builds the HTTP router.
func New(
	sessionHandler *handler.SessionHandler,
	streamWS *handler.StreamWSHandler,
	health *handler.HealthHandler,
) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET(constants.PathHealth, health.Health)
	r.GET(constants.PathReady, health.Ready)

	// REST sessions
	sessions := r.Group("/sessions")
	{
		sessions.POST("", sessionHandler.CreateSession)
		sessions.DELETE("/:id", sessionHandler.DeleteSession)
		sessions.GET("/:id/operators", sessionHandler.GetSessionOperators)
	}

	// WebSocket: /ws/stream/:session_id/:user_id
	r.GET("/ws/stream/:session_id/:user_id", streamWS.ServeWS)

	return r
}
