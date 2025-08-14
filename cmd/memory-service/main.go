// Command memory-service starts the Mycelian Memory HTTP API server.
//
// It loads configuration from environment variables with the
// MEMORY_SERVER_ prefix (see server/internal/config) and runs until
// interrupted. On any startup or runtime error, the process logs the
// error and exits with status 1.
//
// Health endpoint: GET /api/health. For full routing, see
// server/memoryservice.Run.
package main

import (
	"os"

	"github.com/mycelian/mycelian-memory/server/memoryservice"
	"github.com/rs/zerolog/log"
)

// main boots the service via memoryservice.Run and exits non-zero on error.
func main() {
	if err := memoryservice.Run(); err != nil {
		log.Error().Stack().Err(err).Msg("memory-service exited with error")
		os.Exit(1)
	}
}
