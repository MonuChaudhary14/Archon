package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type mockWSConn struct {
	writeChan chan []byte
}

func (m *mockWSConn) WriteMessage(messageType int, data []byte) error {
	m.writeChan <- data
	return nil
}

func (m *mockWSConn) ReadMessage() (int, []byte, error) {
	return 0, nil, errors.New("not implemented")
}

func (m *mockWSConn) Close() error {
	return nil
}

func TestHubRedisPubSub(t *testing.T) {
	rClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rClient.Ping(ctx).Err(); err != nil {
		t.Skip("Skipping Hub test: local Redis not running on localhost:6379")
	}

	hub := NewHub(rClient)
	sessionID := "test-session-pubsub"

	writeChan := make(chan []byte, 1)
	conn := &mockWSConn{writeChan: writeChan}

	hub.Register(sessionID, conn)

	time.Sleep(150 * time.Millisecond)

	payload := []byte("hello from pubsub")
	published := hub.SendMessage(sessionID, payload)
	if !published {
		t.Error("expected SendMessage to return true")
	}

	select {
	case received := <-writeChan:
		if string(received) != string(payload) {
			t.Errorf("expected %s, got %s", payload, received)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for Redis Pub/Sub message to reach WebSocket")
	}

	hub.Unregister(sessionID)

	time.Sleep(100 * time.Millisecond)

	hub.mu.Lock()
	_, exists := hub.connections[sessionID]
	_, cancelExists := hub.cancels[sessionID]
	hub.mu.Unlock()

	if exists {
		t.Error("expected connection to be deleted from hub")
	}
	if cancelExists {
		t.Error("expected cancel func to be deleted from hub")
	}
}
