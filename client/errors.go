package client

import (
	"errors"

	"github.com/mycelian/mycelian-memory/client/internal/types"
)

// ErrBackPressure is returned when the client's internal shard queue is full.
var ErrBackPressure = errors.New("back-pressure (queue full)")

// IsBackPressure reports whether err is a back-pressure error.
func IsBackPressure(err error) bool { return errors.Is(err, ErrBackPressure) }

// Re-export shared SDK error so callers compare against a single symbol.
var ErrNotFound = types.ErrNotFound
