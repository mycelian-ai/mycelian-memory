package main

import (
	"os"

	"github.com/mycelian/mycelian-memory/mcp"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := mcp.RunMCPServer(); err != nil {
		log.Error().Err(err).Msg("MCP server exited with error")
		os.Exit(1)
	}
}
