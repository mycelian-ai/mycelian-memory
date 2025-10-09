package storage

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/health"
	"github.com/rs/zerolog"
)

// StoreHealthChecker monitors store health via periodic probes.
type StoreHealthChecker struct {
	store        Store
	healthy      atomic.Int32
	log          zerolog.Logger
	probeTimeout time.Duration
}

// NewStoreHealthChecker creates a new store health checker.
func NewStoreHealthChecker(store Store, log zerolog.Logger, probeTimeout time.Duration) *StoreHealthChecker {
	return &StoreHealthChecker{store: store, log: log, probeTimeout: probeTimeout}
}

// Name returns the checker name.
func (hc *StoreHealthChecker) Name() string { return "store" }

// IsHealthy returns the cached health status (non-blocking).
func (hc *StoreHealthChecker) IsHealthy() bool { return hc.healthy.Load() == 1 }

// Start begins periodic health checking.
func (hc *StoreHealthChecker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	check := func() {
		to := hc.probeTimeout
		if to <= 0 {
			to = 2 * time.Second
		}
		checkCtx, cancel := context.WithTimeout(ctx, to)
		defer cancel()
		if hc.probe(checkCtx) {
			hc.healthy.Store(1)
		} else {
			hc.healthy.Store(0)
		}
	}

	check()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

// probe executes a simple read to verify database connectivity.
func (hc *StoreHealthChecker) probe(ctx context.Context) bool {
	// Prefer specialized HealthPing if provided by the underlying implementation
	if p, ok := any(hc.store).(health.HealthPinger); ok {
		if err := p.HealthPing(ctx); err != nil {
			hc.log.Error().Stack().Str("checker", hc.Name()).Err(err).Msg("store health check failed")
			return false
		}
		return true
	}

	// Fallback: try a simple read operation using Users().Get
	_, err := hc.store.Users().Get(ctx, "__health_check__")
	if err != nil {
		// ErrNotFound is acceptable - means DB is responsive
		if errors.Is(err, ErrNotFound) {
			return true
		}
		hc.log.Error().Stack().Str("checker", hc.Name()).Err(err).Msg("store health check failed")
		return false
	}
	return true
}
