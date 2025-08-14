package searchindex

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/health"
	"github.com/rs/zerolog"
)

// SearchIndexHealthChecker monitors search index health using an optional
// HealthPinger implemented by the concrete index (e.g., Weaviate, Qdrant).
type SearchIndexHealthChecker struct {
	index        Index
	healthy      atomic.Int32
	log          zerolog.Logger
	probeTimeout time.Duration
}

// NewSearchIndexHealthChecker creates a new search index health checker.
func NewSearchIndexHealthChecker(index Index, log zerolog.Logger, probeTimeout time.Duration) *SearchIndexHealthChecker {
	hc := &SearchIndexHealthChecker{index: index, log: log, probeTimeout: probeTimeout}
	hc.healthy.Store(0) // start unhealthy until first successful probe
	return hc
}

func (hc *SearchIndexHealthChecker) Name() string    { return "searchindex" }
func (hc *SearchIndexHealthChecker) IsHealthy() bool { return hc.healthy.Load() == 1 }

func (hc *SearchIndexHealthChecker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		to := hc.probeTimeout
		if to <= 0 {
			to = 2 * time.Second
		}
		checkCtx, cancel := context.WithTimeout(ctx, to)
		defer cancel()

		ok := true
		if p, okCast := hc.index.(health.HealthPinger); okCast {
			if err := p.HealthPing(checkCtx); err != nil {
				ok = false
				hc.log.Error().Stack().Str("checker", hc.Name()).Err(err).Msg("search index health check failed")
			}
		} else {
			// Fallback: issue a cheap no-op by calling DeleteVault with empty inputs should be no-op
			if err := hc.index.DeleteVault(checkCtx, "", ""); err != nil {
				ok = false
				hc.log.Error().Stack().Str("checker", hc.Name()).Err(err).Msg("search index health check failed")
			}
		}
		if ok {
			hc.healthy.Store(1)
		} else {
			hc.healthy.Store(0)
		}
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
