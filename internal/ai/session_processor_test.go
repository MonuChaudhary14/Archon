package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/MonuChaudhary14/Archon/internal/diagram"
)

type MockWebSocketConnection struct {
	ReadMessageFunc  func() (int, []byte, error)
	WriteMessageFunc func(int, []byte) error
}

func (m *MockWebSocketConnection) ReadMessage() (int, []byte, error) {
	if m.ReadMessageFunc != nil {
		return m.ReadMessageFunc()
	}
	return 0, nil, errors.New("not implemented")
}

func (m *MockWebSocketConnection) WriteMessage(messageType int, data []byte) error {
	if m.WriteMessageFunc != nil {
		return m.WriteMessageFunc(messageType, data)
	}
	return nil
}

func (m *MockWebSocketConnection) Close() error {
	return nil
}

type MockConnectionHub struct {
	RegisterCalled   bool
	UnregisterCalled bool
}

func (m *MockConnectionHub) Register(sessionID string, conn WebSocketConnection) {
	m.RegisterCalled = true
}

func (m *MockConnectionHub) Unregister(sessionID string) {
	m.UnregisterCalled = true
}

func (m *MockConnectionHub) SendMessage(sessionID string, message []byte) bool {
	return true
}

type MockMessageBroker struct {
	PromptPublished bool
	SessionID       string
	Prompt          string
}

func (m *MockMessageBroker) PublishPrompt(sessionID, prompt string) error {
	m.PromptPublished = true
	m.SessionID = sessionID
	m.Prompt = prompt
	return nil
}

func (m *MockMessageBroker) PublishDiagramEvent(sessionID, eventType string, data json.RawMessage) error {
	return nil
}

func (m *MockMessageBroker) PublishEvent(ctx context.Context, key []byte, payload []byte) error {
	return nil
}

type MockChatHistoryRepository struct{}

func (m *MockChatHistoryRepository) GetChatHistory(ctx context.Context, sessionID string) ([]string, error) {
	return []string{
		`{"type":"human","data":{"content":"hello"}}`,
		`{"type":"ai","data":{"content":"hi"}}`,
	}, nil
}

type MockDiagramRepository struct{}

func (m *MockDiagramRepository) SaveNode(ctx context.Context, n diagram.Node) error {
	return nil
}
func (m *MockDiagramRepository) DeleteNode(ctx context.Context, interviewID string, id string) error {
	return nil
}
func (m *MockDiagramRepository) SaveEdge(ctx context.Context, e diagram.Edge) error {
	return nil
}
func (m *MockDiagramRepository) DeleteEdge(ctx context.Context, interviewID string, id string) error {
	return nil
}
func (m *MockDiagramRepository) GetDiagram(ctx context.Context, interviewID string) ([]diagram.Node, []diagram.Edge, error) {
	return nil, nil, nil
}

func TestProcessSession(t *testing.T) {
	broker := &MockMessageBroker{}
	history := &MockChatHistoryRepository{}
	diagRepo := &MockDiagramRepository{}
	hub := &MockConnectionHub{}

	processor := NewSessionProcessor(broker, history, diagRepo, hub)

	callCount := 0
	ws := &MockWebSocketConnection{
		ReadMessageFunc: func() (int, []byte, error) {
			callCount++
			if callCount == 1 {
				msg := WSMessage{
					Type: "chat",
					Data: json.RawMessage(`"design a rate limiter"`),
				}
				bytes, _ := json.Marshal(msg)
				return 1, bytes, nil
			}
			return 0, nil, errors.New("closing connection")
		},
		WriteMessageFunc: func(msgType int, data []byte) error {
			return nil
		},
	}

	err := processor.ProcessSession(context.Background(), "test-session", ws)
	if err == nil || err.Error() != "closing connection" {
		t.Errorf("expected closing connection error, got %v", err)
	}

	if !hub.RegisterCalled {
		t.Error("expected Register to be called")
	}
	if !hub.UnregisterCalled {
		t.Error("expected Unregister to be called")
	}
	if !broker.PromptPublished {
		t.Error("expected broker to publish prompt")
	}
	if broker.Prompt != "design a rate limiter" {
		t.Errorf("expected prompt 'design a rate limiter', got '%s'", broker.Prompt)
	}
}
