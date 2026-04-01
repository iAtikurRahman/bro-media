package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/iAtikurRahman/bro-media/backend/internal/config"
	"github.com/iAtikurRahman/bro-media/backend/internal/signaling"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate against allowed origins
		return true
	},
}

type Handler struct {
	hub *signaling.Hub
	cfg *config.Config
}

func New(hub *signaling.Hub, cfg *config.Config) *Handler {
	return &Handler{hub: hub, cfg: cfg}
}

// ICEServerConfig is returned to clients so they know how to reach STUN/TURN.
type ICEServerConfig struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type ICEServersResponse struct {
	ICEServers []ICEServerConfig `json:"iceServers"`
}

// HandleICEServers returns STUN/TURN configuration to the frontend.
func (h *Handler) HandleICEServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := ICEServersResponse{
		ICEServers: []ICEServerConfig{
			{
				URLs: []string{h.cfg.STUNHost},
			},
			{
				URLs:       []string{fmt.Sprintf("turn:%s:%s", h.cfg.TURNHost, h.cfg.TURNPort)},
				Username:   h.cfg.TURNUser,
				Credential: h.cfg.TURNPassword,
			},
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleHealth provides a basic health check endpoint.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleWebSocket upgrades the HTTP connection to a WebSocket for signaling.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("clientId")
	if clientID == "" {
		http.Error(w, "clientId query parameter is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[handler] websocket upgrade error: %v", err)
		return
	}

	client := signaling.NewClient(clientID, conn, h.hub)
	log.Printf("[handler] new websocket connection: %s", clientID)

	go client.WritePump()
	client.ReadPump(h.onMessage)
}

// onMessage handles incoming signaling messages from a client.
func (h *Handler) onMessage(client *signaling.Client, raw []byte) {
	var msg signaling.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("[handler] invalid message from %s: %v", client.ID, err)
		return
	}

	msg.SenderID = client.ID

	switch msg.Type {
	case "join":
		client.SetRoom(msg.RoomID)
		h.hub.Join(msg.RoomID, client)

	case "offer", "answer", "ice-candidate":
		if msg.TargetID == "" {
			log.Printf("[handler] message type %s from %s missing targetId", msg.Type, client.ID)
			return
		}
		data, _ := json.Marshal(msg)
		h.hub.Route(msg.RoomID, client.ID, msg.TargetID, data)

	case "leave":
		h.hub.Leave(msg.RoomID, client.ID)

	default:
		log.Printf("[handler] unknown message type %s from %s", msg.Type, client.ID)
	}
}
