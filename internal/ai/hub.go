package ai

import (
	"context"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	mu          sync.RWMutex
	connections map[string]WebSocketConnection
	redisClient *redis.Client
	cancels     map[string]context.CancelFunc
}

func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		connections: make(map[string]WebSocketConnection),
		redisClient: redisClient,
		cancels:     make(map[string]context.CancelFunc),
	}
}

func (h *Hub) Register(sessionID string, conn WebSocketConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cancel, exists := h.cancels[sessionID]; exists {
		cancel()
	}

	h.connections[sessionID] = conn

	ctx, cancel := context.WithCancel(context.Background())
	h.cancels[sessionID] = cancel

	go h.subscribeToRedis(ctx, sessionID, conn)
}

func (h *Hub) subscribeToRedis(ctx context.Context, sessionID string, conn WebSocketConnection) {
	pubsub := h.redisClient.Subscribe(ctx, "session:chat:"+sessionID)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			if err != nil {
				log.Printf("Failed to write websocket message from Redis Pub/Sub: %v", err)
				return
			}
		}
	}
}

func (h *Hub) Unregister(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cancel, exists := h.cancels[sessionID]; exists {
		cancel()
		delete(h.cancels, sessionID)
	}
	delete(h.connections, sessionID)
}

func (h *Hub) SendMessage(sessionID string, message []byte) bool {
	ctx := context.Background()
	err := h.redisClient.Publish(ctx, "session:chat:"+sessionID, string(message)).Err()
	if err != nil {
		log.Printf("Failed to publish message to Redis Pub/Sub: %v", err)
		return false
	}
	return true
}
