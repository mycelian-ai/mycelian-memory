# Multi-stage build for MCP server
FROM golang:1.25.2-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# Copy go.work for workspace mode and all referenced go.mod files
COPY go.work ./

# Copy go.mod files for all modules (needed for workspace mode)
COPY pkg/devauth/go.mod pkg/devauth/
COPY client/go.mod client/go.sum client/
COPY server/go.mod server/go.sum server/
COPY mcp/go.mod mcp/go.sum mcp/
COPY cmd/memory-service/go.mod cmd/memory-service/go.sum cmd/memory-service/
COPY cmd/mycelian-mcp-server/go.mod cmd/mycelian-mcp-server/go.sum cmd/mycelian-mcp-server/
COPY cmd/outbox-worker/go.mod cmd/outbox-worker/go.sum cmd/outbox-worker/
COPY tools/mycelianCli/go.mod tools/mycelianCli/go.sum tools/mycelianCli/
COPY tools/mycelian-service-tools/go.mod tools/mycelian-service-tools/go.sum tools/mycelian-service-tools/
COPY tools/invariants-checker/go.mod tools/invariants-checker/go.sum tools/invariants-checker/

# Download dependencies for the modules we'll actually use
RUN cd mcp && go mod download
RUN cd client && go mod download

# Copy source code (only what we need)
COPY mcp/ mcp/
COPY client/ client/
COPY pkg/devauth/ pkg/devauth/
COPY cmd/mycelian-mcp-server/ cmd/mycelian-mcp-server/

# Build using workspace mode
RUN go build -o mycelian-mcp-server ./cmd/mycelian-mcp-server

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/mycelian-mcp-server .

EXPOSE 11546

CMD ["./mycelian-mcp-server"]
