{
  "mcpServers": {
    "synapse-memory": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "--network",
        "host",
        "-e",
        "MEMORY_SERVICE_URL=http://localhost:8080",
        "synapse-mcp-server:dev"
      ]
    }
  }
}