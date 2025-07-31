# Synapse MCP Server

A Go implementation of the Model Context Protocol (MCP) server for Matrix Synapse homeserver administration.

This MCP server provides secure access to Matrix Synapse administrative functions through standardized MCP tools, enabling AI assistants to interact with your Matrix homeserver for user management and administration tasks.

[![Build](https://github.com/synapse/synapse-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/synapse/synapse-mcp-server/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/synapse/synapse-mcp-server)](https://goreportcard.com/report/github.com/synapse/synapse-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

**User Management:**
- **list_users**: List all users from the Synapse homeserver
- **get_user**: Get detailed information about a specific user

**Future Capabilities:**
- Room management
- Federation controls
- Server statistics
- Moderation tools

## Usage

### Prerequisites

- Go 1.24 or later
- Access to a Matrix Synapse server with admin privileges
- MCP-compatible client (e.g., Claude Desktop, Cursor)

### Installation

#### Using Go Install

```bash
go install github.com/synapse/synapse-mcp-server/cmd/synapse-mcp-server@latest
```

#### Building from Source

```bash
git clone https://github.com/synapse/synapse-mcp-server.git
cd synapse-mcp-server
go build -o synapse-mcp-server ./cmd/synapse-mcp-server
```

#### Using Docker

```bash
docker pull ghcr.io/synapse/synapse-mcp-server:latest
```

### Configuration

The server is configured via environment variables:

- `SYNAPSE_URL`: URL of your Synapse server (default: `http://localhost:8080`)

### MCP Client Configuration

#### Claude Desktop

Add the following to your Claude Desktop configuration file:

**Using the binary:**
```json
{
  "mcpServers": {
    "synapse": {
      "command": "synapse-mcp-server",
      "args": [],
      "env": {
        "SYNAPSE_URL": "http://localhost:8080"
      }
    }
  }
}
```

**Using Docker:**
```json
{
  "mcpServers": {
    "synapse": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "-e",
        "SYNAPSE_URL",
        "ghcr.io/synapse/synapse-mcp-server:latest"
      ],
      "env": {
        "SYNAPSE_URL": "http://localhost:8080"
      }
    }
  }
}
```

#### Cursor

Add to your Cursor settings:

```json
{
  "mcpServers": {
    "synapse-mcp-server": {
      "command": "synapse-mcp-server",
      "env": {
        "SYNAPSE_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Available Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `list_users` | List all users from Synapse homeserver | None |
| `get_user` | Get detailed user information | `user_id` (required): Matrix user ID |

### Example Usage

Once configured, you can use the tools in your MCP client:

- "List all users on the Matrix server"
- "Get details for user @admin:localhost"
- "Show me information about user @alice:example.com"

### Get Default Prompts

Retrieve the compiled default prompt templates for a given memory type:

```bash
synapse get-prompts --memory-type chat | jq .
```

Sample output:
```jsonc
{
  "version": "v1",
  "context_summary_rules": "# Context & Summary Rules\n...",
  "templates": {
    "context_prompt": "You are ...",
    "entry_capture_prompt": "When you receive ...",
    "summary_prompt": "Summarise ..."
  }
}
```

These templates can be redirected to a file and customised via the prompt-override CLI commands (see docs).

## Development

### Project Structure

```
synapse-mcp-server/
├── cmd/
│   └── synapse-mcp-server/     # Main application entry point
│       └── main.go
├── internal/                   # Private application code
│   ├── handlers/               # MCP tool handlers
│   │   └── user_handler.go
│   ├── server/                 # Server implementation
│   └── types/                  # Internal types
├── pkg/                        # Public library code
│   ├── config/                 # Configuration management
│   │   └── config.go
│   └── synapse/                # Synapse client library
│       ├── client.go
│       └── types.go
├── docs/                       # Documentation
├── memory-bank/                # Memory bank for development context
├── go.mod
├── go.sum
└── README.md
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Building

```bash
# Build for current platform
go build ./cmd/synapse-mcp-server

# Build for all platforms
make build-all

# Format code
go fmt ./...

# Run linter
golangci-lint run
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/new-feature`)
3. Commit your changes (`git commit -am 'feat(scope): add new feature'`)
4. Push to the branch (`git push origin feature/new-feature`)
5. Create a Pull Request

### Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

- `feat(scope): description` - New features
- `fix(scope): description` - Bug fixes
- `docs(scope): description` - Documentation changes
- `refactor(scope): description` - Code refactoring
- `test(scope): description` - Test additions/modifications
- `chore(scope): description` - Maintenance tasks

## Security Considerations

- Always use secure connections to your Synapse server
- Limit MCP server access to trusted clients only
- Regularly audit user permissions and access logs
- Use service accounts with minimal required permissions

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Matrix.org](https://matrix.org/) for the Matrix protocol and Synapse server
- [Anthropic](https://www.anthropic.com/) for the Model Context Protocol specification
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) for the Go MCP SDK

## Support

- [GitHub Issues](https://github.com/synapse/synapse-mcp-server/issues) for bug reports and feature requests
- [Matrix Community](https://matrix.to/#/#synapse:matrix.org) for community support
- [Documentation](./docs/) for detailed guides and examples

### Context Snapshot API (v3)

The SDK exposes two helper methods for storing and retrieving the *memory-level* context document:

| Method | HTTP | Notes |
|--------|------|-------|
| `PutContext(ctx, userID, memoryID, text)` | `PUT /api/users/{userId}/memories/{memoryId}/contexts` | Enqueued via the shard executor to preserve FIFO ordering. Writes the raw text under `{ "activeContext": "…" }` on the backend. |
| `GetLatestContext(ctx, userID, memoryID)` | `GET /api/users/{userId}/memories/{memoryId}/contexts` | Returns the most-recent snapshot. The local filesystem cache used in earlier versions has been removed.

This decouples context persistence from entry writes and avoids embedding bulky blobs in every `add_entry` request. 