package indexer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// Indexer is the main orchestrator for scanning Spanner and uploading to Waviate.
// For now it only emits a heartbeat every interval – real logic will be added
// in subsequent tasks.

type Indexer struct {
	interval time.Duration
	log      zerolog.Logger

	embedder Embedder
	scanner  Scanner
	uploader *Uploader
	state    *State

	once       bool
	backoffMin time.Duration
	backoffMax time.Duration
}

// New returns a new Indexer with the given configuration.
func New(cfg Config, embedder Embedder, scanner Scanner, uploader *Uploader, state *State, log zerolog.Logger) *Indexer {
	return &Indexer{
		interval:   cfg.Interval,
		log:        log.With().Str("component", "indexer").Logger(),
		embedder:   embedder,
		scanner:    scanner,
		uploader:   uploader,
		state:      state,
		once:       cfg.Once,
		backoffMin: 2 * time.Second,
		backoffMax: 30 * time.Second,
	}
}

// Run blocks until the context is cancelled, emitting a heartbeat at each tick.
func (i *Indexer) Run(ctx context.Context) error {
	ticker := time.NewTicker(i.interval)
	defer ticker.Stop()

	// warm-up embedder
	if i.embedder != nil {
		i.log.Info().Msg("performing embedder warm-up...")
		vec, err := i.embedder.Embed(ctx, "hello")
		if err != nil {
			return fmt.Errorf("embedder warm-up failed: %w", err)
		}
		i.log.Info().Int("vector_dim", len(vec)).Msg("embedder ready")
	}

	entryWM, err := i.state.Load("entries", "global")
	if err != nil {
		i.log.Warn().Err(err).Msg("failed to load entry watermark; starting from zero")
	}
	ctxWM, err := i.state.Load("contexts", "global")
	if err != nil {
		i.log.Warn().Err(err).Msg("failed to load context watermark; starting from zero")
	}

	runCycle := func() error {
		if i.scanner == nil || i.uploader == nil {
			// allow nil components for unit tests
			return nil
		}

		entries, err := i.scanner.FetchEntriesSince(ctx, entryWM, 1000)
		if err != nil {
			return err
		}

		contexts, err := i.scanner.FetchContextsSince(ctx, ctxWM, 1000)
		if err != nil {
			return err
		}

		// Log early diagnostics for this cycle
		i.log.Info().
			Time("entry_watermark", entryWM).
			Time("ctx_watermark", ctxWM).
			Int("entries_found", len(entries)).
			Int("contexts_found", len(contexts)).
			Msg("scan results")

		// Process entries
		if len(entries) > 0 {
			idxEntries := make([]IndexedEntry, 0, len(entries))
			for _, e := range entries {
				text := e.Summary
				if text == "" {
					text = e.RawEntry
				}
				var vec []float32
				if i.embedder != nil {
					v, err := i.embedder.Embed(ctx, text)
					if err != nil {
						i.log.Warn().Err(err).Msg("embedding failed; using empty vector")
					} else {
						vec = v
					}
				}
				idxEntries = append(idxEntries, IndexedEntry{Entry: e, SummaryVector: vec})
			}
			if err := i.uploader.Upsert(ctx, idxEntries); err != nil {
				return fmt.Errorf("upsert entries: %w", err)
			}
			entryWM = entries[len(entries)-1].CreationTime
			_ = i.state.Save("entries", "global", entryWM)
			i.log.Info().Int("entries", len(entries)).Time("entry_watermark", entryWM).Msg("entries uploaded")
		}

		// Process contexts
		if len(contexts) > 0 {
			if err := i.uploader.UpsertContexts(ctx, contexts, i.embedder); err != nil {
				return fmt.Errorf("upsert contexts: %w", err)
			}
			ctxWM = contexts[len(contexts)-1].CreationTime
			_ = i.state.Save("contexts", "global", ctxWM)
			i.log.Info().Int("contexts", len(contexts)).Time("ctx_watermark", ctxWM).Msg("contexts uploaded")
		}
		if len(entries) == 0 && len(contexts) == 0 {
			return nil
		}

		i.log.Info().Msg("cycle complete")
		return nil
	}

	i.log.Info().Msg("indexer started")

	backoff := i.backoffMin

	for {
		if err := runCycle(); err != nil {
			// If the error is due to context cancellation, exit immediately.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				i.log.Info().Err(err).Msg("context cancelled, shutting down indexer")
				return err
			}

			i.log.Error().Err(err).Dur("sleep", backoff).Msg("cycle error, backing off")
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				i.log.Info().Msg("indexer shutting down")
				return ctx.Err()
			}
			if backoff < i.backoffMax {
				backoff *= 2
				if backoff > i.backoffMax {
					backoff = i.backoffMax
				}
			}
		} else {
			backoff = i.backoffMin
		}

		if i.once {
			i.log.Info().Msg("once mode complete; exiting")
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			i.log.Info().Msg("indexer shutting down")
			return ctx.Err()
		}
	}
}
