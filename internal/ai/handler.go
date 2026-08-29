package ai

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/MonuChaudhary14/Archon/internal/diagram"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type SafeWebSocketConn struct {
	mu sync.Mutex
	*websocket.Conn
}

func (s *SafeWebSocketConn) WriteMessage(messageType int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return s.Conn.WriteMessage(messageType, data)
}

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
	hub := NewHub(redisClient)
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

		ws.SetReadLimit(maxMessageSize)
		_ = ws.SetReadDeadline(time.Now().Add(pongWait))
		ws.SetPongHandler(func(string) error {
			_ = ws.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		safeWs := &SafeWebSocketConn{Conn: ws}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		go func() {
			ticker := time.NewTicker(pingPeriod)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := safeWs.WriteMessage(websocket.PingMessage, nil); err != nil {
						_ = safeWs.Close()
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		defer func() {
			_ = safeWs.Close()
		}()

		if err := processor.ProcessSession(c.Request.Context(), sessionID, safeWs); err != nil {
			log.Printf("WebSocket session finished: %v", err)
		}
	}
}
