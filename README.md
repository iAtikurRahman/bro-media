# Bro Media

Peer-to-peer video/audio calling platform built entirely in Go, using WebRTC for media and WebSocket-based signaling.

## Architecture

```
┌─────────────┐     WebSocket      ┌─────────────┐
│  Frontend   │ ◄────────────────► │   Backend   │
│  (Go :3000) │                    │  (Go :8080) │
└──────┬──────┘                    └──────┬──────┘
       │                                  │
       │  WebRTC (P2P or relayed)         │ ICE config
       │                                  │
       └──────────────────────────────────►│
                                   ┌──────┴──────┐
                                   │ TURN Server │
                                   │(coturn:3478)│
                                   └─────────────┘
```

| Service | Technology | Port |
|---------|-----------|------|
| Frontend | Go + embedded HTML/JS/CSS | 3000 |
| Backend | Go (WebSocket signaling) | 8080 |
| TURN Server | coturn (Alpine container) | 3478, 5349 |

## Quick Start (Docker)

```bash
docker-compose up --build
```

Then open **http://localhost:3000** in two browser tabs, enter the same room ID, and start a call.

## Run Locally (without Docker)

### Backend

```bash
cd backend
go run main.go
```

### Frontend

```bash
cd frontend
go run main.go
```

### TURN Server

Use the Docker image or install coturn separately:

```bash
cd turn-server
docker build -t bro-turn .
docker run -p 3478:3478/udp -p 3478:3478/tcp bro-turn
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` (backend) / `3000` (frontend) | Listen port |
| `TURN_HOST` | `turn-server` | TURN server hostname |
| `TURN_PORT` | `3478` | TURN server port |
| `TURN_USER` | `bromedia` | TURN credentials user |
| `TURN_PASSWORD` | `bromedia-secret` | TURN credentials password |
| `BACKEND_URL` | `http://localhost:8080` | Backend API URL (frontend) |
| `WS_URL` | `ws://localhost:8080` | WebSocket URL (frontend) |

## Project Structure

```
bro-media/
├── docker-compose.yml
├── frontend/          # Go web server with embedded templates
│   ├── main.go
│   ├── Dockerfile
│   ├── templates/     # HTML templates
│   └── static/        # CSS & JS assets
├── backend/           # Go signaling server
│   ├── main.go
│   ├── Dockerfile
│   └── internal/
│       ├── config/
│       ├── handler/
│       └── signaling/
└── turn-server/       # coturn Docker wrapper
    ├── Dockerfile
    ├── turnserver.conf
    └── entrypoint.sh
```
