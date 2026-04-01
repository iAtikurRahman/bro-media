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

// Client represents a single WebSocket connection.
type Client struct {
	ID     string
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	roomID string
	once   sync.Once
}

func NewClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:   id,
		conn: conn,
		send: make(chan []byte, 256),
		hub:  hub,
	}
}

// Send queues a message to be written to the WebSocket.
func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		log.Printf("[client] send buffer full for %s, dropping message", c.ID)
	}
}

// SetRoom sets the room this client is joined to.
func (c *Client) SetRoom(roomID string) {
	c.roomID = roomID
}

// Close cleans up the client connection.
func (c *Client) Close() {
	c.once.Do(func() {
		if c.roomID != "" {
			c.hub.Leave(c.roomID, c.ID)
		}
		close(c.send)
		c.conn.Close()
	})
}

// ReadPump reads messages from the WebSocket and routes them via the hub.
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
				log.Printf("[client] write error for %s: %v", c.ID, err)
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
