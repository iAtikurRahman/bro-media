# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /frontend main.go

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
COPY --from=builder /frontend /usr/local/bin/frontend

EXPOSE 3000

ENTRYPOINT ["frontend"]
