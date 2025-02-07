package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients    map[string]*websocket.Conn
	register   chan connection
	unregister chan string
	send       chan bytesData
	exists     chan check
	mu         sync.Mutex
}

type check struct {
	id     string
	result chan bool
}

type connection struct {
	id   string
	conn *websocket.Conn
}

type bytesData struct {
	id   string
	data []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*websocket.Conn),
		register:   make(chan connection),
		unregister: make(chan string),
		send:       make(chan bytesData),
		exists:     make(chan check),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			h.clients[reg.id] = reg.conn
			h.mu.Unlock()

		case id := <-h.unregister:
			h.mu.Lock()
			if conn, exists := h.clients[id]; exists {
				conn.Close()
				delete(h.clients, id)
			}
			h.mu.Unlock()

		case check := <-h.exists:
			h.mu.Lock()
			_, exists := h.clients[check.id]
			h.mu.Unlock()
			check.result <- exists

		case msg := <-h.send:
			h.Send(msg.id, msg.data)
		}
	}
}

func (h *Hub) Send(id string, data []byte) {
	h.mu.Lock()
	conn, exists := h.clients[id]
	h.mu.Unlock()

	if exists {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.mu.Lock()
			conn.Close()
			delete(h.clients, id)
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Exists(id string) bool {
	result := make(chan bool)
	h.exists <- check{id, result}
	return <-result
}
