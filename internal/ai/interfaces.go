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

type PromptPublisher interface {
	PublishPrompt(sessionID, prompt string) error
}

type DiagramEventPublisher interface {
	PublishDiagramEvent(sessionID, eventType string, data json.RawMessage) error
}

type DomainEventPublisher interface {
	PublishEvent(ctx context.Context, topic string, key []byte, payload []byte) error
}

type ChatHistoryRepository interface {
	GetChatHistory(ctx context.Context, sessionID string) ([]string, error)
}
