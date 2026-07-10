package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"devctl/internal/applog"
	"devctl/internal/runtime"
)

var upgrader = websocket.Upgrader{
	// Local-only trust model (SPEC.md §29) — every origin is the same
	// developer's browser talking to their own machine.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub fans runtime events out to every connected WebSocket client. It
// implements runtime.EventPublisher so the runtime manager can publish
// without knowing anything about HTTP/WS.
type Hub struct {
	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*wsClient]struct{})}
}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
}

func (hub *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		applog.Error("ws", "ws upgrade failed: %v", err)
		return
	}

	client := &wsClient{conn: conn, send: make(chan []byte, 64)}
	hub.register(client)

	go hub.writePump(client)
	hub.readPump(client)
}

func (hub *Hub) register(c *wsClient) {
	hub.mu.Lock()
	hub.clients[c] = struct{}{}
	hub.mu.Unlock()
}

func (hub *Hub) unregister(c *wsClient) {
	hub.mu.Lock()
	if _, ok := hub.clients[c]; ok {
		delete(hub.clients, c)
		close(c.send)
	}
	hub.mu.Unlock()
}

// readPump ignores incoming messages (this is a server->client-only feed for
// alpha) but must keep reading so disconnects and control frames (pings) are
// handled, per gorilla/websocket's documented pattern.
func (hub *Hub) readPump(c *wsClient) {
	defer func() {
		hub.unregister(c)
		_ = c.conn.Close()
	}()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (hub *Hub) writePump(c *wsClient) {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// Broadcast sends data to every connected client without blocking the
// caller; a client whose send buffer is full is dropped rather than stalling
// the runtime manager on a slow reader.
func (hub *Hub) Broadcast(data []byte) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for c := range hub.clients {
		select {
		case c.send <- data:
		default:
			delete(hub.clients, c)
			close(c.send)
		}
	}
}

// Publish implements runtime.EventPublisher.
func (hub *Hub) Publish(evt runtime.AppEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		applog.Error("ws", "ws: marshal event: %v", err)
		return
	}
	hub.Broadcast(data)
}
