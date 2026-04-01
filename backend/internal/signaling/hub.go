package signaling

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Message represents a signaling message exchanged between clients.
type Message struct {
	Type      string          `json:"type"`
	RoomID    string          `json:"roomId,omitempty"`
	SenderID  string          `json:"senderId,omitempty"`
	TargetID  string          `json:"targetId,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Text      string          `json:"text,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
}

// Room holds clients in a single call room.
type Room struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// Hub manages all active rooms, routes signaling, and tracks online users.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	users map[string]*Client // online users by username
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
		users: make(map[string]*Client),
	}
}

// Register adds a client to the global online users list.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	if old, exists := h.users[client.ID]; exists {
		old.ForceClose()
	}
	h.users[client.ID] = client
	h.mu.Unlock()
	log.Printf("[hub] user %s online (total: %d)", client.ID, len(h.users))
	h.BroadcastOnlineUsers()
}

// Unregister removes a client from the online users list.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if existing, ok := h.users[client.ID]; ok && existing == client {
		delete(h.users, client.ID)
	}
	h.mu.Unlock()
	log.Printf("[hub] user %s offline", client.ID)
	h.BroadcastOnlineUsers()
}

// OnlineUsers returns a list of online usernames.
func (h *Hub) OnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := make([]string, 0, len(h.users))
	for u := range h.users {
		users = append(users, u)
	}
	return users
}

// SendToUser sends a message directly to a specific online user.
func (h *Hub) SendToUser(username string, data []byte) {
	h.mu.RLock()
	client, ok := h.users[username]
	h.mu.RUnlock()
	if ok {
		client.Send(data)
	}
}

// BroadcastOnlineUsers sends the current online users list to every connected client.
func (h *Hub) BroadcastOnlineUsers() {
	users := h.OnlineUsers()
	payload, _ := json.Marshal(users)
	msg, _ := json.Marshal(Message{
		Type:    "online-users",
		Payload: payload,
	})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.users {
		client.Send(msg)
	}
}

// Join adds a client to a room and notifies existing peers.
func (h *Hub) Join(roomID string, client *Client) {
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	if !exists {
		room = &Room{clients: make(map[string]*Client)}
		h.rooms[roomID] = room
	}
	h.mu.Unlock()

	room.mu.Lock()
	defer room.mu.Unlock()

	// Notify existing peers about the new participant
	for id, peer := range room.clients {
		peerMsg, _ := json.Marshal(Message{
			Type:     "peer-joined",
			RoomID:   roomID,
			SenderID: client.ID,
		})
		peer.Send(peerMsg)

		// Tell the new client about existing peers
		existingMsg, _ := json.Marshal(Message{
			Type:     "peer-joined",
			RoomID:   roomID,
			SenderID: id,
		})
		client.Send(existingMsg)
	}

	room.clients[client.ID] = client
	log.Printf("[hub] %s joined room %s (total: %d)", client.ID, roomID, len(room.clients))
}

// Leave removes a client from a room and notifies remaining peers.
func (h *Hub) Leave(roomID string, clientID string) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()
	if !exists {
		return
	}

	room.mu.Lock()
	delete(room.clients, clientID)
	remaining := len(room.clients)
	room.mu.Unlock()

	// Notify remaining peers
	leaveMsg, _ := json.Marshal(Message{
		Type:     "peer-left",
		RoomID:   roomID,
		SenderID: clientID,
	})
	h.Broadcast(roomID, clientID, leaveMsg)

	// Clean up empty rooms
	if remaining == 0 {
		h.mu.Lock()
		delete(h.rooms, roomID)
		h.mu.Unlock()
	}

	log.Printf("[hub] %s left room %s (remaining: %d)", clientID, roomID, remaining)
}

// Route sends a message to a specific target client in a room.
func (h *Hub) Route(roomID, senderID, targetID string, data []byte) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()
	if !exists {
		return
	}

	room.mu.RLock()
	target, ok := room.clients[targetID]
	room.mu.RUnlock()
	if ok {
		target.Send(data)
	}
}

// Broadcast sends a message to all clients in a room except the sender.
func (h *Hub) Broadcast(roomID, senderID string, data []byte) {
	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()
	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for id, client := range room.clients {
		if id != senderID {
			client.Send(data)
		}
	}
}

// BroadcastChat sends a chat message to all clients in a room (including sender).
func (h *Hub) BroadcastChat(roomID, senderID, text string) {
	msg, _ := json.Marshal(Message{
		Type:      "chat",
		RoomID:    roomID,
		SenderID:  senderID,
		Text:      text,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	h.mu.RLock()
	room, exists := h.rooms[roomID]
	h.mu.RUnlock()
	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()
	for _, client := range room.clients {
		client.Send(msg)
	}
}
