FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go module files first for better cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source
COPY . .

# Build the indexer-prototype binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o indexer-prototype ./cmd/indexer-prototype

# ---- Runtime image ----
FROM alpine:latest

# Install CA certificates for outbound HTTPS calls (e.g., Weaviate/Ollama over HTTPS)
RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/indexer-prototype .

# No ports exposed; service runs in background polling loop
CMD ["./indexer-prototype"] 