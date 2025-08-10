package main

import (
    "os"

    "github.com/mycelian/mycelian-memory/server/memoryservice"
    "github.com/rs/zerolog/log"
)

func main() {
    if err := memoryservice.Run(); err != nil {
        log.Error().Err(err).Msg("memory-service exited with error")
        os.Exit(1)
    }
}


