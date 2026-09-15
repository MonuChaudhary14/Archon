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
	connections map[string]map[WebSocketConnection]context.CancelFunc
	redisClient *redis.Client
}

func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		connections: make(map[string]map[WebSocketConnection]context.CancelFunc),
		redisClient: redisClient,
	}
}

func (h *Hub) Register(sessionID string, conn WebSocketConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.connections[sessionID]; !exists {
		h.connections[sessionID] = make(map[WebSocketConnection]context.CancelFunc)
	}

	if cancel, exists := h.connections[sessionID][conn]; exists {
		cancel()
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	h.connections[sessionID][conn] = cancelFunc

	log.Printf("[Hub.Register] Session %s registered (active conns for session: %d)", sessionID, len(h.connections[sessionID]))
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

func (h *Hub) Unregister(sessionID string, conn WebSocketConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, exists := h.connections[sessionID]
	if !exists {
		return
	}

	if cancel, ok := conns[conn]; ok {
		cancel()
		delete(conns, conn)
	}

	if len(conns) == 0 {
		delete(h.connections, sessionID)
	}
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
