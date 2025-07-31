# Build stage
FROM golang:1.24.4-alpine AS builder

# Install git for go modules
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the MCP server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o synapse-mcp-server ./cmd/synapse-mcp-server

# Build the Synapse CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o synapse ./cmd/synapse

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -g 1001 -S synapse && \
    adduser -u 1001 -S synapse -G synapse

WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/synapse-mcp-server .
COPY --from=builder /app/synapse .

# Change ownership to non-root user
RUN chown synapse:synapse synapse-mcp-server

# Switch to non-root user
USER synapse

# Expose port (if needed for future HTTP transport)
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8081/ || exit 1

# Set default environment variables
ENV MEMORY_SERVICE_URL=http://localhost:8080

# Run the binary
ENTRYPOINT ["./synapse-mcp-server"] 