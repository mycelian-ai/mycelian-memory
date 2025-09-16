# Multi-stage build for MCP server
FROM golang:1.24.6-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

# Copy the entire repository to leverage go.work workspace
COPY . .

# Build the MCP server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mycelian-mcp-server ./cmd/mycelian-mcp-server

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/mycelian-mcp-server .

EXPOSE 11546

CMD ["./mycelian-mcp-server"]
