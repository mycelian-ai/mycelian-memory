package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/rs/zerolog"

	"github.com/mycelian/mycelian-memory/server/internal/search"
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
SELECT id, op, payload, aggregate_id
FROM outbox
WHERE status = 'pending' AND next_attempt_at <= now()
ORDER BY id ASC
FOR UPDATE SKIP LOCKED
LIMIT $1`

	markDoneSQL = `UPDATE outbox SET status='done', update_time=now() WHERE id=$1`

	markFailedSQL = `
UPDATE outbox
SET attempt_count = attempt_count + 1,
    next_attempt_at = now() + make_interval(secs => LEAST(POWER(2, attempt_count+1), 300)),
    update_time = now()
WHERE id=$1`
)

// Config controls batch size and polling cadence.
type Config struct {
	PostgresDSN string        // currently unused here (DB is injected), kept for symmetry with main
	BatchSize   int           // number of rows to lease per cycle
	Interval    time.Duration // poll interval
}

// Worker processes outbox rows and applies them to the vector store.
type Worker struct {
	db       *sql.DB
	log      zerolog.Logger
	embedder search.Embedder
	index    search.Searcher
	cfg      Config
}

// NewWorker constructs a Worker from dependencies.
func NewWorker(db *sql.DB, emb search.Embedder, idx search.Searcher, cfg Config, log zerolog.Logger) *Worker {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
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
	id          int64
	op          string
	aggregateID string
	payload     map[string]interface{}
}

func (w *Worker) processOnce(ctx context.Context) error {
	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	jobs, err := w.leaseBatch(ctx, tx, w.cfg.BatchSize)
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return tx.Commit()
	}

	for _, j := range jobs {
		if err := w.handle(ctx, j); err != nil {
			if e := w.markFailed(ctx, tx, j.id, err); e != nil {
				w.log.Error().Err(e).Int64("id", j.id).Msg("markFailed error")
			}
			continue
		}
		if e := w.markDone(ctx, tx, j.id); e != nil {
			w.log.Error().Err(e).Int64("id", j.id).Msg("markDone error")
		}
	}

	return tx.Commit()
}

// leaseBatch locks and returns up to batchSize ready outbox rows.
func (w *Worker) leaseBatch(ctx context.Context, tx *sql.Tx, batchSize int) ([]job, error) {
	rows, err := tx.QueryContext(ctx, selectReadyRowsSQL, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []job
	for rows.Next() {
		var j job
		var raw []byte
		if err := rows.Scan(&j.id, &j.op, &raw, &j.aggregateID); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &j.payload); err != nil {
			// Poison pill: mark failed so it backs off and won’t hot-loop
			_ = w.markFailed(ctx, tx, j.id, errors.New("bad payload"))
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// handle executes the outbox operation.
func (w *Worker) handle(ctx context.Context, j job) error {
	switch j.op {
	case OpUpsertEntry:
		text := preferredText(j.payload, "summary", "rawEntry")
		vec, err := w.embed(text, ctx)
		if err != nil {
			return err
		}
		return w.index.UpsertEntry(ctx, j.aggregateID, vec, j.payload)
	case OpDeleteEntry:
		return w.index.DeleteEntry(ctx, stringField(j.payload, "userId"), j.aggregateID)
	case OpUpsertContext:
		text := stringField(j.payload, "context")
		vec, err := w.embed(text, ctx)
		if err != nil {
			return err
		}
		return w.index.UpsertContext(ctx, j.aggregateID, vec, j.payload)
	case OpDeleteContext:
		return w.index.DeleteContext(ctx, stringField(j.payload, "userId"), j.aggregateID)
	default:
		return fmt.Errorf("unknown op: %s", j.op)
	}
}

func (w *Worker) markDone(ctx context.Context, tx *sql.Tx, id int64) error {
	_, err := tx.ExecContext(ctx, markDoneSQL, id)
	return err
}

func (w *Worker) markFailed(ctx context.Context, tx *sql.Tx, id int64, cause error) error {
	_, err := tx.ExecContext(ctx, markFailedSQL, id)
	return err
}

// embed wraps the embedder to keep callers simple and guards nil embedder usage.
func (w *Worker) embed(text string, ctx context.Context) ([]float32, error) {
	if w.embedder == nil {
		// Allow operation without vectors (indexer may fill later); return empty vector
		return nil, nil
	}
	return w.embedder.Embed(ctx, text)
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
