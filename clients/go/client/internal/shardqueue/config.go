package shardqueue

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config groups all tunables.  Values are taken from environment variables with
// the prefix "SQ_". Example: SQ_SHARDS=8 SQ_QUEUE_SIZE=256 .
type Config struct {
	Shards         int           `envconfig:"SHARDS"          default:"4"`
	QueueSize      int           `envconfig:"QUEUE_SIZE"      default:"128"`
	EnqueueTimeout time.Duration `envconfig:"ENQUEUE_TIMEOUT" default:"100ms"`

	// ErrorHandler is called synchronously after a Job returns a non‑nil error.
	// Leave nil if you do not care.
	ErrorHandler func(error) `envconfig:"-"`

	MaxAttempts int           `envconfig:"MAX_ATTEMPTS"   default:"8"`
	BaseBackoff time.Duration `envconfig:"BASE_BACKOFF"    default:"100ms"`
	MaxInterval time.Duration `envconfig:"MAX_INTERVAL"    default:"20s"`
}

// LoadConfig populates Config from environment variables (prefix SQ_).
func LoadConfig() (Config, error) {
	var c Config
	return c, envconfig.Process("SQ", &c)
}
