package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iAtikurRahman/bro-media/backend/internal/auth"
	"github.com/iAtikurRahman/bro-media/backend/internal/config"
	"github.com/iAtikurRahman/bro-media/backend/internal/signaling"
	"github.com/iAtikurRahman/bro-media/backend/internal/store"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler struct {
	hub   *signaling.Hub
	cfg   *config.Config
	store *store.Store
}

func New(hub *signaling.Hub, cfg *config.Config, s *store.Store) *Handler {
	return &Handler{hub: hub, cfg: cfg, store: s}
}

// ---- Auth Endpoints ----

type signupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// HandleSignup registers a new user.
func (h *Handler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 || len(req.Password) < 6 || req.Email == "" {
		jsonError(w, "username (min 3), email, and password (min 6) required", http.StatusBadRequest)
		return
	}

	// Allow only alphanumeric + underscore
	for _, c := range req.Username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			jsonError(w, "username may only contain letters, numbers, and underscores", http.StatusBadRequest)
			return
		}
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	user := &store.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	if err := h.store.CreateUser(user); err != nil {
		if err == store.ErrUserExists {
			jsonError(w, "username or email already taken", http.StatusConflict)
			return
		}
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(req.Username, h.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse{Token: token, Username: req.Username})
}

// HandleLogin authenticates an existing user.
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.store.GetUser(req.Username)
	if err != nil {
		jsonError(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		jsonError(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(req.Username, h.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse{Token: token, Username: req.Username})
}

// ---- ICE Servers ----

type ICEServerConfig struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// HandleICEServers returns STUN/TURN configuration to the frontend.
func (h *Handler) HandleICEServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"iceServers": []ICEServerConfig{
			{URLs: []string{h.cfg.STUNHost}},
			{
				URLs:       []string{fmt.Sprintf("turn:%s:%s", h.cfg.TURNHost, h.cfg.TURNPort)},
				Username:   h.cfg.TURNUser,
				Credential: h.cfg.TURNPassword,
			},
		},
	})
}

// ---- Health ----

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---- WebSocket ----

// HandleWebSocket upgrades an authenticated HTTP connection to WebSocket.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		if ah := r.Header.Get("Authorization"); strings.HasPrefix(ah, "Bearer ") {
			tokenStr = strings.TrimPrefix(ah, "Bearer ")
		}
	}
	if tokenStr == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	claims, err := auth.ValidateToken(tokenStr, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[handler] websocket upgrade error: %v", err)
		return
	}

	client := signaling.NewClient(claims.Username, conn, h.hub)
	h.hub.Register(client)
	log.Printf("[handler] websocket connected: %s", claims.Username)

	go client.WritePump()
	client.ReadPump(h.onMessage)
}

// ---- Message Router ----

func (h *Handler) onMessage(client *signaling.Client, raw []byte) {
	var msg signaling.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("[handler] invalid message from %s: %v", client.ID, err)
		return
	}

	msg.SenderID = client.ID

	switch msg.Type {
	case "join":
		if msg.RoomID == "" {
			return
		}
		client.AddRoom(msg.RoomID)
		h.hub.Join(msg.RoomID, client)

	case "leave":
		if msg.RoomID == "" {
			return
		}
		client.RemoveRoom(msg.RoomID)
		h.hub.Leave(msg.RoomID, client.ID)

	case "offer", "answer", "ice-candidate":
		if msg.TargetID == "" || msg.RoomID == "" {
			return
		}
		data, _ := json.Marshal(msg)
		h.hub.Route(msg.RoomID, client.ID, msg.TargetID, data)

	case "chat":
		if msg.RoomID == "" || msg.Text == "" {
			return
		}
		h.hub.BroadcastChat(msg.RoomID, client.ID, msg.Text)

	case "call-user":
		if msg.TargetID == "" {
			return
		}
		roomID := privateRoomID(client.ID, msg.TargetID)
		callMsg, _ := json.Marshal(signaling.Message{
			Type:     "call-incoming",
			RoomID:   roomID,
			SenderID: client.ID,
		})
		h.hub.SendToUser(msg.TargetID, callMsg)

	case "call-accept":
		if msg.TargetID == "" || msg.RoomID == "" {
			return
		}
		acceptMsg, _ := json.Marshal(signaling.Message{
			Type:     "call-accepted",
			RoomID:   msg.RoomID,
			SenderID: client.ID,
		})
		h.hub.SendToUser(msg.TargetID, acceptMsg)

	case "call-reject":
		if msg.TargetID == "" {
			return
		}
		rejectMsg, _ := json.Marshal(signaling.Message{
			Type:     "call-rejected",
			SenderID: client.ID,
		})
		h.hub.SendToUser(msg.TargetID, rejectMsg)

	default:
		log.Printf("[handler] unknown message type %s from %s", msg.Type, client.ID)
	}
}

func privateRoomID(a, b string) string {
	if a < b {
		return "dm-" + a + "-" + b
	}
	return "dm-" + b + "-" + a
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
