package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/iAtikurRahman/bro-media/backend/internal/config"
	"github.com/iAtikurRahman/bro-media/backend/internal/handler"
	"github.com/iAtikurRahman/bro-media/backend/internal/signaling"
)

func main() {
	cfg := config.Load()
	hub := signaling.NewHub()
	h := handler.New(hub, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.HandleHealth)
	mux.HandleFunc("/api/ice-servers", h.HandleICEServers)
	mux.HandleFunc("/ws", h.HandleWebSocket)

	// Wrap with CORS middleware
	wrapped := corsMiddleware(cfg.AllowedOrigins, mux)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("[main] signaling server starting on %s", addr)
	if err := http.ListenAndServe(addr, wrapped); err != nil {
		log.Fatalf("[main] server failed: %v", err)
	}
}

// corsMiddleware adds CORS headers for allowed origins.
func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if origin == "" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
