package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/rs/zerolog"

	emb "github.com/mycelian/mycelian-memory/server/internal/embeddings"
	"github.com/mycelian/mycelian-memory/server/internal/searchindex"
)

// Operation names stored in outbox.op (idempotent targets)
const (
	OpUpsertEntry   = "upsert_entry"
	OpDeleteEntry   = "delete_entry"
	OpUpsertContext = "upsert_context"
	OpDeleteContext = "delete_context"
)

// SQL statements kept as constants for clarity and reuse
const (
	selectReadyRowsSQL = `
SELECT id, op, payload, aggregate_id, attempt_count, next_attempt_at, leased_until
FROM outbox
WHERE status = 'pending' AND next_attempt_at <= now()
ORDER BY id ASC
FOR UPDATE SKIP LOCKED
LIMIT $1`

	resetExpiredLeasesSQL = `
UPDATE outbox
SET status='pending',
	leased_until=NULL,
	update_time=now()
WHERE status='processing' AND leased_until IS NOT NULL AND leased_until <= now()`

	markProcessingSQL = `
UPDATE outbox
SET status='processing',
	leased_until = now() + make_interval(secs => $2),
	update_time = now()
WHERE id=$1`

	markDoneSQL = `UPDATE outbox SET status='done', leased_until=NULL, update_time=now() WHERE id=$1`

	markFailedSQL = `
UPDATE outbox
SET attempt_count = attempt_count + 1,
	status='pending',
	leased_until=NULL,
	next_attempt_at = now() + make_interval(secs => LEAST(POWER(2, attempt_count+1), $2)),
	update_time = now()
WHERE id=$1`
)

// Config controls batch size and polling cadence.
type Config struct {
	PostgresDSN   string        // currently unused here (DB is injected), kept for symmetry with main
	BatchSize     int           // number of rows to lease per cycle
	Interval      time.Duration // poll interval
	EmbedTimeout  time.Duration
	IndexTimeout  time.Duration
	LeaseDuration time.Duration
	BackoffCap    time.Duration
}

// Worker processes outbox rows and applies them to the vector store.
type Worker struct {
	db       *sql.DB
	log      zerolog.Logger
	embedder emb.EmbeddingProvider
	index    searchindex.Index
	cfg      Config
}

// NewWorker constructs a Worker from dependencies.
func NewWorker(db *sql.DB, emb emb.EmbeddingProvider, idx searchindex.Index, cfg Config, log zerolog.Logger) *Worker {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}
	if cfg.EmbedTimeout <= 0 {
		cfg.EmbedTimeout = 12 * time.Second
	}
	if cfg.IndexTimeout <= 0 {
		cfg.IndexTimeout = 5 * time.Second
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = 30 * time.Second
	}
	if cfg.BackoffCap <= 0 {
		cfg.BackoffCap = 60 * time.Second
	}
	return &Worker{db: db, log: log, embedder: emb, index: idx, cfg: cfg}
}

// Run starts the polling loop until ctx is canceled.
func (w *Worker) Run(ctx context.Context) error {
	w.log.Info().Int("batch", w.cfg.BatchSize).Dur("interval", w.cfg.Interval).Msg("outbox worker starting")
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info().Msg("outbox worker stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := w.processOnce(ctx); err != nil {
				// Log and continue; per-row backoff prevents hot-looping
				w.log.Error().Err(err).Msg("outbox processOnce")
			}
		}
	}
}

type job struct {
	id           int64
	op           string
	aggregateID  string
	payload      map[string]interface{}
	attemptCount int
	nextAttempt  time.Time
	leasingEnds  time.Time
}

func (w *Worker) processOnce(ctx context.Context) error {
	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, resetExpiredLeasesSQL); err != nil {
		return err
	}

	jobs, err := w.leaseBatch(ctx, tx, w.cfg.BatchSize, w.cfg.LeaseDuration)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if len(jobs) == 0 {
		return nil
	}

	for _, j := range jobs {
		jobStart := time.Now()
		if err := w.handle(ctx, j); err != nil {
			nextDelay := w.failureBackoffDuration(j.attemptCount + 1)
			nextETA := time.Now().Add(nextDelay)
			// Surface per-row failures with enough context to debug
			w.log.Error().
				Err(err).
				Int64("id", j.id).
				Str("op", j.op).
				Str("aggregate_id", j.aggregateID).
				Int("attempt", j.attemptCount+1).
				Dur("elapsed", time.Since(jobStart)).
				Dur("next_delay", nextDelay).
				Time("leased_until", j.leasingEnds).
				Time("prev_next_attempt_at", j.nextAttempt).
				Time("next_attempt_at_est", nextETA).
				Msg("outbox handle error; marking failed")

			if e := w.markFailed(ctx, j.id); e != nil {
				w.log.Error().Err(e).Int64("id", j.id).Msg("markFailed error")
			}
			continue
		}
		if e := w.markDone(ctx, j.id); e != nil {
			w.log.Error().Err(e).Int64("id", j.id).Msg("markDone error")
		}
	}

	return nil
}

// leaseBatch locks and returns up to batchSize ready outbox rows.
func (w *Worker) leaseBatch(ctx context.Context, tx *sql.Tx, batchSize int, leaseDuration time.Duration) ([]job, error) {
	rows, err := tx.QueryContext(ctx, selectReadyRowsSQL, batchSize)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if leaseDuration <= 0 {
		leaseDuration = 30 * time.Second
	}
	leaseSeconds := float64(leaseDuration / time.Second)
	if leaseSeconds <= 0 {
		leaseSeconds = 30
	}
	backoffCapSeconds := float64(w.cfg.BackoffCap / time.Second)
	if backoffCapSeconds <= 0 {
		backoffCapSeconds = 60
	}

	var jobs []job
	for rows.Next() {
		var j job
		var raw []byte
		var leaseUntil sql.NullTime
		if err := rows.Scan(&j.id, &j.op, &raw, &j.aggregateID, &j.attemptCount, &j.nextAttempt, &leaseUntil); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &j.payload); err != nil {
			// Poison pill: mark failed so it backs off and won’t hot-loop
			_, _ = tx.ExecContext(ctx, markFailedSQL, j.id, backoffCapSeconds)
			continue
		}
		if leaseUntil.Valid {
			j.leasingEnds = leaseUntil.Time
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range jobs {
		if _, err := tx.ExecContext(ctx, markProcessingSQL, jobs[i].id, leaseSeconds); err != nil {
			return nil, err
		}
		jobs[i].leasingEnds = time.Now().Add(leaseDuration)
	}

	return jobs, nil
}

// handle executes the outbox operation.
func (w *Worker) handle(ctx context.Context, j job) error {
	attempt := j.attemptCount + 1
	w.log.Info().Str("op", j.op).Str("aggregateId", j.aggregateID).Int64("id", j.id).Int("attempt", attempt).Msg("processing outbox job")

	switch j.op {
	case OpUpsertEntry:
		text := preferredText(j.payload, "summary", "rawEntry")
		// Skip embedding/upsert when there is no usable text
		if strings.TrimSpace(text) == "" {
			w.log.Warn().Str("entryId", j.aggregateID).Int("attempt", attempt).Msg("empty entry text; skipping indexing and marking done")
			return nil
		}
		w.log.Debug().Str("text", text).Str("entryId", j.aggregateID).Int("attempt", attempt).Msg("upserting entry")
		vec, err := w.embed(ctx, text)
		if err != nil {
			w.log.Error().Err(err).Str("entryId", j.aggregateID).Int("attempt", attempt).Msg("embedding failed")
			return err
		}
		w.log.Debug().Int("vectorLength", len(vec)).Str("entryId", j.aggregateID).Int("attempt", attempt).Msg("embedding generated")
		normalizeEntryTags(j.payload)
		err = w.withIndexTimeout(ctx, func(idxCtx context.Context) error {
			return w.index.UpsertEntry(idxCtx, j.aggregateID, vec, j.payload)
		})
		if err != nil {
			if isAlreadyExists(err) {
				w.log.Info().Str("entryId", j.aggregateID).Int("attempt", attempt).Msg("entry already in index; marking done")
				return nil
			}
			w.log.Error().Err(err).Str("entryId", j.aggregateID).Int("attempt", attempt).Msg("upsert entry failed")
			return err
		}
		w.log.Info().Str("entryId", j.aggregateID).Int("attempt", attempt).Msg("entry upserted successfully")
		return nil
	case OpDeleteEntry:
		return w.withIndexTimeout(ctx, func(idxCtx context.Context) error {
			return w.index.DeleteEntry(idxCtx, stringField(j.payload, "actorId"), j.aggregateID)
		})
	case OpUpsertContext:
		text := stringField(j.payload, "context")
		// Skip embedding/upsert when there is no usable text
		if strings.TrimSpace(text) == "" {
			w.log.Warn().Str("contextId", j.aggregateID).Int("attempt", attempt).Msg("empty context text; skipping indexing and marking done")
			return nil
		}
		vec, err := w.embed(ctx, text)
		if err != nil {
			return err
		}
		if err := w.withIndexTimeout(ctx, func(idxCtx context.Context) error {
			return w.index.UpsertContext(idxCtx, j.aggregateID, vec, j.payload)
		}); err != nil {
			if isAlreadyExists(err) {
				w.log.Info().Str("contextId", j.aggregateID).Int("attempt", attempt).Msg("context already in index; marking done")
				return nil
			}
			return err
		}
		return nil
	case OpDeleteContext:
		return w.withIndexTimeout(ctx, func(idxCtx context.Context) error {
			return w.index.DeleteContext(idxCtx, stringField(j.payload, "actorId"), j.aggregateID)
		})
	default:
		return fmt.Errorf("unknown op: %s", j.op)
	}
}

func (w *Worker) markDone(ctx context.Context, id int64) error {
	_, err := w.db.ExecContext(ctx, markDoneSQL, id)
	return err
}

func (w *Worker) markFailed(ctx context.Context, id int64) error {
	capSeconds := float64(w.cfg.BackoffCap / time.Second)
	if capSeconds <= 0 {
		capSeconds = 60
	}
	_, err := w.db.ExecContext(ctx, markFailedSQL, id, capSeconds)
	return err
}

// embed wraps the embedder with a timeout to keep callers simple.
// Embedder is guaranteed to be non-nil after startup validation.
func (w *Worker) embed(ctx context.Context, text string) ([]float32, error) {
	if w.cfg.EmbedTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.cfg.EmbedTimeout)
		defer cancel()
	}
	return w.embedder.Embed(ctx, text)
}

func (w *Worker) withIndexTimeout(ctx context.Context, fn func(context.Context) error) error {
	if w.cfg.IndexTimeout <= 0 {
		return fn(ctx)
	}
	ctx, cancel := context.WithTimeout(ctx, w.cfg.IndexTimeout)
	defer cancel()
	return fn(ctx)
}

func stringField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		}
	}
	return ""
}

func preferredText(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if s := stringField(m, k); s != "" {
			return s
		}
	}
	return ""
}

// normalizeEntryTags ensures the payload["tags"] is a []string of tag keys
// so that Weaviate schema (text[]) can store and filter via ContainsAny.
func normalizeEntryTags(m map[string]interface{}) {
	v, ok := m["tags"]
	if !ok || v == nil {
		return
	}
	switch t := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(t))
		for k, val := range t {
			switch vv := val.(type) {
			case bool:
				if vv {
					keys = append(keys, k)
				}
			case string:
				if strings.EqualFold(vv, "true") {
					keys = append(keys, k)
				}
			default:
				// ignore non-bool marker values
			}
		}
		m["tags"] = keys
	case []interface{}:
		// convert to []string
		keys := make([]string, 0, len(t))
		for _, it := range t {
			if s, ok := it.(string); ok {
				keys = append(keys, s)
			}
		}
		m["tags"] = keys
	}
}

// isAlreadyExists returns true when the vector index reports that the
// object ID already exists. We use a substring match to avoid coupling
// to a specific client error type.
func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "status code: 422")
}

// failureBackoffDuration mirrors the SQL backoff calculation capped via configuration.
func (w *Worker) failureBackoffDuration(nextAttemptCount int) time.Duration {
	if nextAttemptCount <= 0 {
		return 0
	}
	capSeconds := w.cfg.BackoffCap.Seconds()
	if capSeconds <= 0 {
		capSeconds = 60
	}
	pow := math.Pow(2, float64(nextAttemptCount))
	if pow > capSeconds {
		pow = capSeconds
	}
	return time.Duration(pow) * time.Second
}
