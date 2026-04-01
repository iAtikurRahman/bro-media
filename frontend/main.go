package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	port := getEnv("PORT", "3000")
	backendURL := getEnv("BACKEND_URL", "http://localhost:8080")
	wsURL := getEnv("WS_URL", "ws://localhost:8080")

	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("[frontend] failed to parse templates: %v", err)
	}

	mux := http.NewServeMux()

	// Serve static assets (CSS, JS)
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// Main page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := map[string]string{
			"BackendURL": backendURL,
			"WsURL":      wsURL,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("[frontend] template error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("[frontend] starting on :%s (backend=%s)", port, backendURL)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("[frontend] server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
