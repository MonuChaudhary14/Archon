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
			if err := json.Unmarshal([]byte(msgStr), &msgMap); err == nil {
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

				if data, ok := msgMap["data"].(map[string]interface{}); ok {
					if content, ok := data["content"].(string); ok {
						payload := map[string]string{
							"role":    role,
							"content": content,
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
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			if err := p.promptPub.PublishPrompt(sessionID, string(msg)); err != nil {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("Error sending message to processing queue"))
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
			if err := p.promptPub.PublishPrompt(sessionID, prompt); err != nil {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("Error sending message to processing queue"))
			}

		case "node_added", "node_updated":
			var node diagram.Node
			if err := json.Unmarshal(wsMsg.Data, &node); err == nil {
				node.InterviewID = sessionID
				if err := p.diagRepo.SaveNode(ctx, node); err != nil {
					log.Printf("Failed to save node: %v", err)
					_ = conn.WriteMessage(websocket.TextMessage, []byte("Error saving diagram node"))
				} else {
					if err := p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data); err != nil {
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
			if err := json.Unmarshal(wsMsg.Data, &payload); err == nil {
				if err := p.diagRepo.DeleteNode(ctx, sessionID, payload.ID); err != nil {
					log.Printf("Failed to delete node: %v", err)
					_ = conn.WriteMessage(websocket.TextMessage, []byte("Error deleting diagram node"))
				} else {
					if err := p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data); err != nil {
						log.Printf("Failed to publish diagram event: %v", err)
					}
				}
			}

		case "edge_added", "edge_updated":
			var edge diagram.Edge
			if err := json.Unmarshal(wsMsg.Data, &edge); err == nil {
				edge.InterviewID = sessionID
				if err := p.diagRepo.SaveEdge(ctx, edge); err != nil {
					log.Printf("Failed to save edge: %v", err)
					_ = conn.WriteMessage(websocket.TextMessage, []byte("Error saving diagram edge"))
				} else {
					if err := p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data); err != nil {
						log.Printf("Failed to publish diagram event: %v", err)
					}
				}
			}

		case "edge_deleted":
			var payload struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(wsMsg.Data, &payload); err == nil {
				if err := p.diagRepo.DeleteEdge(ctx, sessionID, payload.ID); err != nil {
					log.Printf("Failed to delete edge: %v", err)
					_ = conn.WriteMessage(websocket.TextMessage, []byte("Error deleting diagram edge"))
				} else {
					if err := p.diagramPub.PublishDiagramEvent(sessionID, wsMsg.Type, wsMsg.Data); err != nil {
						log.Printf("Failed to publish diagram event: %v", err)
					}
				}
			}

		default:
			log.Printf("Unknown WebSocket message type: %s", wsMsg.Type)
		}
	}
}
