# Multi-stage build for memory-service
FROM golang:1.25.2-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better layer caching
COPY server/go.mod server/go.sum server/
COPY pkg/devauth/go.mod pkg/devauth/
COPY cmd/memory-service/go.mod cmd/memory-service/go.sum cmd/memory-service/

# Download dependencies (cached unless go.mod changes)
# Use GOWORK=off to avoid needing the full workspace
RUN cd server && GOWORK=off go mod download
RUN cd cmd/memory-service && GOWORK=off go mod download

# Copy source code
COPY server/ server/
COPY pkg/devauth/ pkg/devauth/
COPY cmd/memory-service/ cmd/memory-service/

# Build within the module directory to avoid go.work workspace coupling
WORKDIR /app/cmd/memory-service
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/memory-service .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/
COPY --from=builder /app/memory-service .

EXPOSE 11545 9090
CMD ["./memory-service"]
