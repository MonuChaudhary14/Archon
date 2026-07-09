package ai

import(
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct{
	mu sync.RWMutex
	connections map[string]*websocket.Conn
}

func NewHub() *Hub{
	return &Hub{
		connections : make(map[string]*websocket.Conn),
	}
}

func (h *Hub) Register(sessionID string, conn *websocket.Conn){
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[sessionID] = conn
}

func (h *Hub) Unregister(sessionID string){
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, sessionID)
}

func (h *Hub) SendMessage(sessionId string, message []byte) bool{
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conn, exists := h.connections[sessionId]; exists{
		conn.WriteMessage(websocket.TextMessage, message)
		return true
	}
	return false
}

