package health

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// HealthChecker is implemented by component-level checkers (store, search, embedder).
type HealthChecker interface {
	Name() string
	IsHealthy() bool
	Start(ctx context.Context, interval time.Duration)
}

// ServiceHealthChecker aggregates component checkers into a single service health flag.
type ServiceHealthChecker struct {
	healthy atomic.Int32
	deps    []HealthChecker
	log     zerolog.Logger
}

func NewServiceHealthChecker(log zerolog.Logger, deps ...HealthChecker) *ServiceHealthChecker {
	h := &ServiceHealthChecker{deps: deps, log: log}
	h.healthy.Store(0)
	return h
}

// IsHealthy returns cached service health.
func (h *ServiceHealthChecker) IsHealthy() bool { return h.healthy.Load() == 1 }

// Start periodically evaluates dependency health and updates the service flag.
func (h *ServiceHealthChecker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	prev := int32(0)
	eval := func() {
		all := true
		for _, c := range h.deps {
			if !c.IsHealthy() {
				all = false
			}
		}
		if all {
			h.healthy.Store(1)
		} else {
			h.healthy.Store(0)
		}
		cur := h.healthy.Load()
		if cur != prev {
			if cur == 1 {
				h.log.Info().Msg("service health: UP")
			} else {
				h.log.Error().Stack().Msg("service health: DOWN")
			}
			prev = cur
		}
	}

	eval()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eval()
		}
	}
}
