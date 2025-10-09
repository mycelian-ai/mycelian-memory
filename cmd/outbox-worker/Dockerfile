# Multi-stage build for outbox-worker
FROM golang:1.25.2-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better layer caching
COPY server/go.mod server/go.sum server/
COPY pkg/devauth/go.mod pkg/devauth/
COPY cmd/outbox-worker/go.mod cmd/outbox-worker/go.sum cmd/outbox-worker/

# Download dependencies (cached unless go.mod changes)
# Use GOWORK=off to avoid needing the full workspace
RUN cd server && GOWORK=off go mod download
RUN cd cmd/outbox-worker && GOWORK=off go mod download

# Copy source code
COPY server/ server/
COPY pkg/devauth/ pkg/devauth/
COPY cmd/outbox-worker/ cmd/outbox-worker/

# Build
WORKDIR /app/cmd/outbox-worker
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /app/outbox-worker .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/
COPY --from=builder /app/outbox-worker ./outbox-worker

ENTRYPOINT ["./outbox-worker"]
