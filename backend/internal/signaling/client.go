package signaling

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024 // 64KB
)

// Client represents a single WebSocket connection tied to a user.
type Client struct {
	ID     string
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	rooms  map[string]bool
	mu     sync.Mutex
	closed bool
}

func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:    id,
		conn:  conn,
		send:  make(chan []byte, 256),
		hub:   hub,
		rooms: make(map[string]bool),
	}
}

// Send queues a message to be written to the WebSocket.
func (c *Client) Send(msg []byte) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	select {
	case c.send <- msg:
	default:
		log.Printf("[client] send buffer full for %s, dropping message", c.ID)
	}
}

// AddRoom tracks a room this client has joined.
func (c *Client) AddRoom(roomID string) {
	c.mu.Lock()
	c.rooms[roomID] = true
	c.mu.Unlock()
}

// RemoveRoom stops tracking a room.
func (c *Client) RemoveRoom(roomID string) {
	c.mu.Lock()
	delete(c.rooms, roomID)
	c.mu.Unlock()
}

// ForceClose shuts down a connection when the same user reconnects.
func (c *Client) ForceClose() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	close(c.send)
	c.conn.Close()
}

// Close cleans up the client, leaving all rooms and unregistering.
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	rooms := make([]string, 0, len(c.rooms))
	for r := range c.rooms {
		rooms = append(rooms, r)
	}
	c.mu.Unlock()

	for _, roomID := range rooms {
		c.hub.Leave(roomID, c.ID)
	}
	c.hub.Unregister(c)
	close(c.send)
	c.conn.Close()
}

// ReadPump reads messages from the WebSocket and routes them via the handler.
func (c *Client) ReadPump(handler func(client *Client, msg []byte)) {
	defer c.Close()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[client] read error for %s: %v", c.ID, err)
			}
			return
		}
		handler(c, message)
	}
}

// WritePump writes messages from the send channel to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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
