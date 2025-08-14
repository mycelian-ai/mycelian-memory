package embeddings

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/health"
	"github.com/rs/zerolog"
)

// ProviderHealthChecker monitors an embeddings provider by calling Embed.
type ProviderHealthChecker struct {
	provider     EmbeddingProvider
	healthy      atomic.Int32
	log          zerolog.Logger
	probeTimeout time.Duration
}

func NewProviderHealthChecker(p EmbeddingProvider, log zerolog.Logger, probeTimeout time.Duration) *ProviderHealthChecker {
	hc := &ProviderHealthChecker{provider: p, log: log, probeTimeout: probeTimeout}
	hc.healthy.Store(0) // start unhealthy until first successful probe
	return hc
}

func (c *ProviderHealthChecker) Name() string    { return "embedder" }
func (c *ProviderHealthChecker) IsHealthy() bool { return c.healthy.Load() == 1 }

func (c *ProviderHealthChecker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		to := c.probeTimeout
		if to <= 0 {
			to = 2 * time.Second
		}
		checkCtx, cancel := context.WithTimeout(ctx, to)
		defer cancel()
		// Prefer specialized HealthPing if available
		if p, ok := any(c.provider).(health.HealthPinger); ok {
			if err := p.HealthPing(checkCtx); err != nil {
				c.healthy.Store(0)
				c.log.Error().Stack().Str("checker", c.Name()).Err(err).Msg("embedder health check failed")
				return
			}
			c.healthy.Store(1)
			return
		}
		// Fallback: attempt a simple embedding
		vec, err := c.provider.Embed(checkCtx, "health-check")
		if err != nil || len(vec) == 0 {
			c.healthy.Store(0)
			c.log.Error().Stack().Str("checker", c.Name()).Err(err).Msg("embedder health check failed")
			return
		}
		c.healthy.Store(1)
	}

	run()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
