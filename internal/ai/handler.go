package ai

import (
	"context"
	"log"
	"net/http"

	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/MonuChaudhary14/Archon/internal/diagram"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Setup(
	router *gin.RouterGroup,
	redisClient *redis.Client,
	diagRepo diagram.Repository,
	jwtSecret string,
	userRepo auth.UserRepository,
	verifyOwner func(context.Context, int, string) (bool, error),
) *KafkaService {
	hub := NewHub()
	kafkaSvc := NewKafkaService(hub)

	go kafkaSvc.StartConsuming()

	historyRepo := NewRedisHistoryRepository(redisClient)
	processor := NewSessionProcessor(kafkaSvc, historyRepo, diagRepo, hub)

	router.GET("/ai/chat", auth.AuthMiddleware(jwtSecret, userRepo), ChatHandler(processor, verifyOwner))

	return kafkaSvc
}

// ChatHandler godoc
// @Summary      Connect to AI Chat WebSocket session
// @Description  Establishes a bidirectional WebSocket connection to stream chat messages and diagram events for a specific interview session.
// @Tags         ai
// @Param        session_id query string true "Interview Session ID"
// @Router       /ai/chat [get]
func ChatHandler(
	processor SessionProcessor,
	verifyOwner func(context.Context, int, string) (bool, error),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("session_id")

		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "session_id required",
			})
			return
		}

		userIDVal, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: user ID not found in context"})
			return
		}

		userIDUint, ok := userIDVal.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error: invalid user ID format"})
			return
		}
		userID := int(userIDUint)

		isOwner, err := verifyOwner(c.Request.Context(), userID, sessionID)
		if err != nil || !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: you do not have access to this interview session"})
			return
		}

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("WebSocket upgrade failed:", err)
			return
		}
		defer func() {
			_ = ws.Close()
		}()

		if err := processor.ProcessSession(c.Request.Context(), sessionID, ws); err != nil {
			log.Printf("WebSocket session finished: %v", err)
		}
	}
}
