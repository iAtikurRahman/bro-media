# Bro Media – TURN Server

coturn wrapped in an Alpine Docker image with runtime env-var configuration.

## Run

```bash
docker build -t bro-turn .
docker run -p 3478:3478/udp -p 3478:3478/tcp bro-turn
```

Credentials default to `bromedia` / `bromedia-secret`, override via `TURN_USER` / `TURN_PASSWORD` env vars.
