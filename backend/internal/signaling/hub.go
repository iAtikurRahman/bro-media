package signaling

import (
	"encoding/json"
	"log"
	"sync"
)

// Message represents a signaling message exchanged between clients.
type Message struct {
	Type     string          `json:"type"` // "offer", "answer", "ice-candidate", "join", "leave"
	RoomID   string          `json:"roomId"`
	SenderID string          `json:"senderId,omitempty"`
	TargetID string          `json:"targetId,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

// Room holds clients in a single call room.
type Room struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// Hub manages all active rooms and routes signaling messages.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
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
	log.Printf("[hub] client %s joined room %s (total: %d)", client.ID, roomID, len(room.clients))
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

	log.Printf("[hub] client %s left room %s (remaining: %d)", clientID, roomID, remaining)
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
