// Package ws contains the WebSocket hub that pushes alerts to every
// connected browser.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second // max time to write one message
	pongWait  = 60 * time.Second // close the connection if no pong arrives in this time
	pingEvery = 30 * time.Second // how often we ping the browser
)

// The frontend is hosted on a different domain than the API, so we accept any origin.
// In production this should be limited to the real frontend URL.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// client is one connected browser tab.
type client struct {
	conn *websocket.Conn
	send chan []byte // messages waiting to be written to this browser
}

// Hub keeps track of all connected clients.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]bool
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]bool)}
}

// ServeWS upgrades an HTTP request to a WebSocket and keeps it open
// until the browser disconnects.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	c := &client{conn: conn, send: make(chan []byte, 16)}
	h.add(c)

	go c.writeLoop()
	c.readLoop(h) // blocks until the client goes away
}

// Broadcast sends a message (any value that can be turned into JSON) to all clients.
// It never blocks: if a client is too slow, that client is dropped.
func (h *Hub) Broadcast(message any) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("broadcast: cannot encode message: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// The client's buffer is full, so it is not reading. Drop it.
			h.dropLocked(c)
		}
	}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	h.dropLocked(c)
	h.mu.Unlock()
}

// dropLocked removes a client. The caller must already hold h.mu.
// Closing the channel tells writeLoop to stop.
func (h *Hub) dropLocked(c *client) {
	if h.clients[c] {
		delete(h.clients, c)
		close(c.send)
	}
}

// readLoop only exists to notice disconnects and to handle pong replies.
// The frontend does not send us any messages.
func (c *client) readLoop(h *Hub) {
	defer func() {
		h.remove(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writeLoop sends queued messages and regular pings to the browser.
func (c *client) writeLoop() {
	ticker := time.NewTicker(pingEvery)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed our channel: say goodbye and stop.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
