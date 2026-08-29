package ai

import (
	"context"
	"encoding/json"
)

type WebSocketConnection interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

type ConnectionHub interface {
	Register(sessionID string, conn WebSocketConnection)
	Unregister(sessionID string)
	SendMessage(sessionID string, message []byte) bool
}

type MessageBroker interface {
	PublishPrompt(sessionID, prompt string) error
	PublishDiagramEvent(sessionID, eventType string, data json.RawMessage) error
	PublishEvent(ctx context.Context, key []byte, payload []byte) error
}

type ChatHistoryRepository interface {
	GetChatHistory(ctx context.Context, sessionID string) ([]string, error)
}
