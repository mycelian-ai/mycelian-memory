package config

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger configures zerolog for text-based output with no coloring.
func InitLogger() {
	// Configure zerolog for text-based output with no coloring
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02 15:04:05",
		NoColor:    true,
	})
}

// SetLogLevel sets the global log level for zerolog.
func SetLogLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}
