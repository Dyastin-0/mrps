package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	clients    map[string]*client
	register   chan connection
	unregister chan string
	exists     chan check
	mu         sync.Mutex
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type check struct {
	id     string
	result chan bool
}

type connection struct {
	id   string
	conn *websocket.Conn
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*client),
		register:   make(chan connection),
		unregister: make(chan string),
		exists:     make(chan check),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			if _, exists := h.clients[reg.id]; exists {
				h.mu.Unlock()
				reg.conn.Close()
				continue
			}

			c := &client{
				conn: reg.conn,
				send: make(chan []byte, 256),
			}
			h.clients[reg.id] = c
			h.mu.Unlock()

			go h.writeWorker(reg.id, c)

		case id := <-h.unregister:
			h.mu.Lock()
			if c, exists := h.clients[id]; exists {
				close(c.send)
				c.conn.Close()
				delete(h.clients, id)
			}
			h.mu.Unlock()

		case check := <-h.exists:
			h.mu.Lock()
			_, exists := h.clients[check.id]
			h.mu.Unlock()
			check.result <- exists
		}
	}
}

func (h *Hub) writeWorker(id string, c *client) {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			h.mu.Lock()
			c.conn.Close()
			delete(h.clients, id)
			h.mu.Unlock()
			break
		}
	}
}

func (h *Hub) Send(id string, data []byte) {
	h.mu.Lock()
	c, exists := h.clients[id]
	h.mu.Unlock()

	if exists {
		select {
		case c.send <- data:
		default:
			h.mu.Lock()
			c.conn.Close()
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
