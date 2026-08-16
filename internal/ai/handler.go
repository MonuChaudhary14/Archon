package ai

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

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

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func Setup(router *gin.RouterGroup, redisClient *redis.Client, diagRepo diagram.Repository) *KafkaService {
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

			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				if err := kafkaSvc.PublishPrompt(sessionID, string(msg)); err != nil {
					ws.WriteMessage(websocket.TextMessage, []byte("Error sending message to processing queue"))
				}
				continue
			}

			switch wsMsg.Type {
			case "chat":
				var prompt string
				if err := json.Unmarshal(wsMsg.Data, &prompt); err != nil {
					var objMap map[string]interface{}
					if err2 := json.Unmarshal(wsMsg.Data, &objMap); err2 == nil {
						if content, ok := objMap["content"].(string); ok {
							prompt = content
						} else {
							prompt = string(wsMsg.Data)
						}
					} else {
						prompt = string(wsMsg.Data)
					}
				}
				if err := kafkaSvc.PublishPrompt(sessionID, prompt); err != nil {
					ws.WriteMessage(websocket.TextMessage, []byte("Error sending message to processing queue"))
				}
			case "node_added", "node_updated":
				var node diagram.Node
				if err := json.Unmarshal(wsMsg.Data, &node); err == nil {
					node.InterviewID = sessionID
					if err := diagRepo.SaveNode(context.Background(), node); err != nil {
						log.Printf("Failed to save node: %v", err)
						ws.WriteMessage(websocket.TextMessage, []byte("Error saving diagram node"))
					} else {
						kafkaSvc.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					}
				} else {
					log.Printf("Failed to unmarshal node: %v", err)
				}
			case "node_deleted":
				var payload struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(wsMsg.Data, &payload); err == nil {
					if err := diagRepo.DeleteNode(context.Background(), sessionID, payload.ID); err != nil {
						log.Printf("Failed to delete node: %v", err)
						ws.WriteMessage(websocket.TextMessage, []byte("Error deleting diagram node"))
					} else {
						kafkaSvc.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					}
				}
			case "edge_added", "edge_updated":
				var edge diagram.Edge
				if err := json.Unmarshal(wsMsg.Data, &edge); err == nil {
					edge.InterviewID = sessionID
					if err := diagRepo.SaveEdge(context.Background(), edge); err != nil {
						log.Printf("Failed to save edge: %v", err)
						ws.WriteMessage(websocket.TextMessage, []byte("Error saving diagram edge"))
					} else {
						kafkaSvc.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					}
				}
			case "edge_deleted":
				var payload struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(wsMsg.Data, &payload); err == nil {
					if err := diagRepo.DeleteEdge(context.Background(), sessionID, payload.ID); err != nil {
						log.Printf("Failed to delete edge: %v", err)
						ws.WriteMessage(websocket.TextMessage, []byte("Error deleting diagram edge"))
					} else {
						kafkaSvc.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					}
				}
			default:
				log.Printf("Unknown WebSocket message type: %s", wsMsg.Type)
			}
		}
	})

	return kafkaSvc
}
