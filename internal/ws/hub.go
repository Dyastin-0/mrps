package ws

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Hub struct {
	clients    map[string]*client
	register   chan connection
	unregister chan string
	exists     chan check
	mu         sync.Mutex
}

type client struct {
	conn    *websocket.Conn
	send    chan []byte
	recv    chan []byte
	closech chan bool
}

type check struct {
	id     string
	result chan bool
}

type connection struct {
	id     string
	conn   *websocket.Conn
	closed chan bool
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*client),
		register:   make(chan connection),
		unregister: make(chan string),
		exists:     make(chan check),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			if c, exists := h.clients[reg.id]; exists {
				h.mu.Unlock()
				c.conn.Close()
				close(c.send)
				close(c.recv)
				continue
			}

			c := &client{
				conn:    reg.conn,
				send:    make(chan []byte, 256),
				recv:    make(chan []byte, 256),
				closech: reg.closed,
			}
			h.clients[reg.id] = c
			h.mu.Unlock()

			go h.writeWorker(reg.id, c)
			go h.readWorker(reg.id, c)

		case id := <-h.unregister:
			h.mu.Lock()
			if c, exists := h.clients[id]; exists {
				close(c.send)
				close(c.recv)
				c.conn.Close()
				delete(h.clients, id)
				c.closech <- true
			}
			h.mu.Unlock()

		case check := <-h.exists:
			h.mu.Lock()
			_, exists := h.clients[check.id]
			h.mu.Unlock()
			check.result <- exists

		case <-ctx.Done():
			log.Info().Str("status", "stopping").Msg("websocket")
			return
		}
	}
}

func (h *Hub) writeWorker(id string, c *client) {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Error().Str("type", "writer").Err(err).Str("client", "..."+id[max(0, len(id)-10):]).Msg("websocket")
			h.unregister <- id
			return
		}
	}
}

func (h *Hub) readWorker(id string, c *client) {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Error().Str("type", "reader").Err(err).Str("client", "..."+id[max(0, len(id)-10):]).Msg("websocket")
			h.unregister <- id
			return
		}

		select {
		case c.recv <- msg:
		default:
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
		}
	}
}

func (h *Hub) Exists(id string) bool {
	result := make(chan bool)
	h.exists <- check{id, result}
	return <-result
}

func (h *Hub) Get(id string) (*websocket.Conn, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	c, exists := h.clients[id]
	if !exists {
		return nil, false
	}
	return c.conn, true
}

func (h *Hub) Listen(id string) (<-chan []byte, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	c, exists := h.clients[id]
	if !exists {
		return nil, false
	}

	return c.recv, true
}
