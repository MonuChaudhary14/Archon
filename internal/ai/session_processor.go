package ai

import (
	"context"
	"encoding/json"
	"log"

	"github.com/MonuChaudhary14/Archon/internal/diagram"
	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type SessionProcessor interface {
	ProcessSession(ctx context.Context, sessionID string, conn WebSocketConnection) error
}

type sessionProcessor struct {
	promptPub    PromptPublisher
	diagramPub   DiagramEventPublisher
	historyStore ChatHistoryRepository
	diagRepo     diagram.Repository
	hub          ConnectionHub
}

func NewSessionProcessor(promptPub PromptPublisher, diagramPub DiagramEventPublisher, historyStore ChatHistoryRepository, diagRepo diagram.Repository, hub ConnectionHub) SessionProcessor {
	return &sessionProcessor{
		promptPub:    promptPub,
		diagramPub:   diagramPub,
		historyStore: historyStore,
		diagRepo:     diagRepo,
		hub:          hub,
	}
}

func (p *sessionProcessor) ProcessSession(ctx context.Context, sessionID string, conn WebSocketConnection) error {
	p.hub.Register(sessionID, conn)
	defer p.hub.Unregister(sessionID)

	history, err := p.historyStore.GetChatHistory(ctx, sessionID)
	if err == nil {
		for _, msgStr := range history {
			var msgMap map[string]interface{}
			err := json.Unmarshal([]byte(msgStr), &msgMap)
			if err == nil {
				var role string
				msgType, _ := msgMap["type"].(string)
				switch msgType {
				case "human":
					role = "user"
				case "ai":
					role = "ai"
				default:
					continue
				}

				data, ok := msgMap["data"].(map[string]interface{})
				if ok {
					content, ok := data["content"].(string)
					if ok {
						payload := map[string]interface{}{
							"type": "chat",
							"data": map[string]string{
								"role":    role,
								"content": content,
							},
						}
						payloadBytes, err := json.Marshal(payload)
						if err == nil {
							_ = conn.WriteMessage(websocket.TextMessage, payloadBytes)
						}
					}
				}
			}
		}
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var wsMsg WSMessage
		err = json.Unmarshal(msg, &wsMsg)
		if err != nil {
			p.sendThinkingStatus(conn)
			err = p.promptPub.PublishPrompt(sessionID, string(msg))
			if err != nil {
				p.sendErrorMessage(conn, "Error sending message to processing queue")
			}
			continue
		}

		switch wsMsg.Type {
		case "chat":
			var prompt string
			err := json.Unmarshal(wsMsg.Data, &prompt)
			if err != nil {
				var objMap map[string]interface{}
				err2 := json.Unmarshal(wsMsg.Data, &objMap)
				if err2 == nil {
					content, ok := objMap["content"].(string)
					if ok {
						prompt = content
					} else {
						prompt = string(wsMsg.Data)
					}
				} else {
					prompt = string(wsMsg.Data)
				}
			}
			p.sendThinkingStatus(conn)
			err = p.promptPub.PublishPrompt(sessionID, prompt)
			if err != nil {
				p.sendErrorMessage(conn, "Error sending message to processing queue")
			}

		case "node_added", "node_updated":
			var node diagram.Node
			err := json.Unmarshal(wsMsg.Data, &node)
			if err == nil {
				node.InterviewID = sessionID
				err = p.diagRepo.SaveNode(ctx, node)
				if err != nil {
					log.Printf("Failed to save node: %v", err)
					p.sendErrorMessage(conn, "Error saving diagram node")
				} else {
					err = p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					if err != nil {
						log.Printf("Failed to publish diagram event: %v", err)
					}
				}
			} else {
				log.Printf("Failed to unmarshal node: %v", err)
			}

		case "node_deleted":
			var payload struct {
				ID string `json:"id"`
			}
			err := json.Unmarshal(wsMsg.Data, &payload)
			if err == nil {
				err = p.diagRepo.DeleteNode(ctx, sessionID, payload.ID)
				if err != nil {
					log.Printf("Failed to delete node: %v", err)
					p.sendErrorMessage(conn, "Error deleting diagram node")
				} else {
					err = p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					if err != nil {
						log.Printf("Failed to publish diagram event: %v", err)
					}
				}
			}

		case "edge_added", "edge_updated":
			var edge diagram.Edge
			err := json.Unmarshal(wsMsg.Data, &edge)
			if err == nil {
				edge.InterviewID = sessionID
				err = p.diagRepo.SaveEdge(ctx, edge)
				if err != nil {
					log.Printf("Failed to save edge: %v", err)
					p.sendErrorMessage(conn, "Error saving diagram edge")
				} else {
					err = p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					if err != nil {
						log.Printf("Failed to publish diagram event: %v", err)
					}
				}
			}

		case "edge_deleted":
			var payload struct {
				ID string `json:"id"`
			}
			err := json.Unmarshal(wsMsg.Data, &payload)
			if err == nil {
				err = p.diagRepo.DeleteEdge(ctx, sessionID, payload.ID)
				if err != nil {
					log.Printf("Failed to delete edge: %v", err)
					p.sendErrorMessage(conn, "Error deleting diagram edge")
				} else {
					err = p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data)
					if err != nil {
						log.Printf("Failed to publish diagram event: %v", err)
					}
				}
			}

		case "ping":
			pongPayload, _ := json.Marshal(map[string]string{
				"type": "pong",
			})
			_ = conn.WriteMessage(websocket.TextMessage, pongPayload)

		default:
			log.Printf("Unknown WebSocket message type: %s", wsMsg.Type)
		}
	}
}

func (p *sessionProcessor) sendThinkingStatus(conn WebSocketConnection) {
	payload, err := json.Marshal(map[string]interface{}{
		"type": "status",
		"data": map[string]string{
			"status": "thinking",
			"role":   "ai",
		},
	})
	if err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}
}

func (p *sessionProcessor) sendErrorMessage(conn WebSocketConnection, message string) {
	payload, err := json.Marshal(map[string]interface{}{
		"type": "error",
		"data": map[string]string{
			"message": message,
		},
	})
	if err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}
}
