# Bro Media – Backend

Go WebSocket signaling server for WebRTC call setup.

## Endpoints

- `GET /health` – health check
- `GET /api/ice-servers` – returns STUN/TURN config
- `GET /ws?clientId=<id>` – WebSocket signaling

## Run

```bash
go run main.go
```

Listens on :8080 by default.
