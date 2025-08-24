package main

import (
	"os"

	"github.com/mycelian/mycelian-memory/server/outboxworker"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := outboxworker.Run(); err != nil {
		log.Error().Err(err).Msg("outbox-worker exited with error")
		os.Exit(1)
	}
}
