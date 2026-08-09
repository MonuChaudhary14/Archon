package ai

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Setup(router *gin.RouterGroup, redisClient *redis.Client) *KafkaService {
	hub := NewHub()
	kafkaSvc := NewKafkaService(hub)

	go kafkaSvc.StartConsuming()

	router.GET("/ai/chat", func(c *gin.Context) {
		sessionID := c.Query("session_id")

		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "session_id required",
			})
			return
		}

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Println("WebSocket upgrade failed:", err)
			return
		}
		defer ws.Close()

		hub.Register(sessionID, ws)
		defer hub.Unregister(sessionID)

		historyKeys, err := redisClient.LRange(context.Background(), "message_store:"+sessionID, 0, -1).Result()
		if err == nil {
			for _, msgStr := range historyKeys {
				var msgMap map[string]interface{}
				if err := json.Unmarshal([]byte(msgStr), &msgMap); err == nil {
					if msgType, ok := msgMap["type"].(string); ok && msgType == "ai" {
						if data, ok := msgMap["data"].(map[string]interface{}); ok {
							if content, ok := data["content"].(string); ok {
								ws.WriteMessage(websocket.TextMessage, []byte(content))
							}
						}
					}
				}
			}
		}

		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}
			if err := kafkaSvc.PublishPrompt(sessionID, string(msg)); err != nil {
				ws.WriteMessage(websocket.TextMessage, []byte("Error sending message to processing queue"))
			}
		}
	})

	return kafkaSvc
}

