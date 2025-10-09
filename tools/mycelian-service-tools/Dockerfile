# Multi-stage build for mycelian-service-tools
FROM golang:1.25.2-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better layer caching
COPY tools/mycelian-service-tools/go.mod tools/mycelian-service-tools/go.sum tools/mycelian-service-tools/

# Download dependencies (cached unless go.mod changes)
# Use GOWORK=off to avoid needing the full workspace
RUN cd tools/mycelian-service-tools && GOWORK=off go mod download

# Copy source code
COPY tools/mycelian-service-tools/ tools/mycelian-service-tools/

# Build
WORKDIR /app/tools/mycelian-service-tools
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /app/mycelian-service-tools .

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/mycelian-service-tools ./mycelian-service-tools

RUN apk add --no-cache curl

ENTRYPOINT ["./mycelian-service-tools"]
