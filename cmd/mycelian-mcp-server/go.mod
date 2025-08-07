module github.com/mycelian/mycelian-memory/cmd/mycelian-mcp-server

go 1.24.5

require (
    github.com/mycelian/mycelian-memory/mcp v0.0.0
    github.com/rs/zerolog v1.34.0
)

replace github.com/mycelian/mycelian-memory/mcp => ../../mcp
