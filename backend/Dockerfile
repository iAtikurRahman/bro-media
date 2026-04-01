# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /backend main.go

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
RUN mkdir -p /data
COPY --from=builder /backend /usr/local/bin/backend

EXPOSE 8080

ENTRYPOINT ["backend"]
