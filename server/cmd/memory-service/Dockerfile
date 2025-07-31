# Multi-stage build for memory service  
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go modules files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the memory service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o memory-service ./cmd/memory-service

# Final stage - minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/memory-service .

# Expose port
EXPOSE 8080

# (Removed Docker-level health check to avoid container restarts when dependencies are temporarily unavailable)

# Run the binary
CMD ["./memory-service"] 