package indexer

import (
	"context"
	"encoding/json"
	"os"
	"time"

	platformdb "github.com/mycelian/mycelian-memory/server/internal/platform/database"

	"cloud.google.com/go/spanner"
	"github.com/rs/zerolog"
	"google.golang.org/api/iterator"
)

// Entry is the minimal projection needed by the indexer loop.
// Additional fields can be added later as required.
type Entry struct {
	UserID       string
	MemoryID     string
	CreationTime time.Time
	EntryID      string
	RawEntry     string
	Summary      string
	Metadata     map[string]interface{}
	Tags         map[string]interface{}
}

// Unique string identifier for the entry suitable for Waviate object ID.
func (e Entry) ID() string { return e.EntryID }

// ContextSnapshot represents one row from MemoryContexts for indexing
type ContextSnapshot struct {
	UserID       string
	MemoryID     string
	ContextID    string
	CreationTime time.Time
	Text         string // flattened / minified JSON context
}

// Scanner interface for fetching data from Spanner.
type Scanner interface {
	FetchEntriesSince(ctx context.Context, since time.Time, limit int) ([]Entry, error)
	FetchContextsSince(ctx context.Context, since time.Time, limit int) ([]ContextSnapshot, error)
	Close() error
}

// SpannerScanner implements the Scanner interface using Spanner.
type SpannerScanner struct {
	client *spanner.Client
	log    zerolog.Logger
}

// NewScanner initialises a Spanner client using the logical identifiers in cfg.
// If cfg.SpannerEmulatorHost is set, the function ensures the environment
// variable SPANNER_EMULATOR_HOST is exported before dialing.
func NewScanner(ctx context.Context, cfg Config, log zerolog.Logger) (Scanner, error) {
	if cfg.SpannerEmulatorHost != "" {
		_ = os.Setenv("SPANNER_EMULATOR_HOST", cfg.SpannerEmulatorHost)
	}

	client, err := platformdb.NewSpannerClient(ctx, platformdb.SpannerConfig{
		ProjectID:  cfg.SpannerProjectID,
		InstanceID: cfg.SpannerInstanceID,
		DatabaseID: cfg.SpannerDatabaseID,
	})
	if err != nil {
		return nil, err
	}

	return &SpannerScanner{client: client, log: log.With().Str("component", "scanner").Logger()}, nil
}

// Close releases underlying Spanner resources.
func (s *SpannerScanner) Close() error {
	s.client.Close()
	return nil
}

// FetchSince returns up to `limit` entries created strictly after `since`.
// Results are ordered ascending by CreationTime (oldest first).
func (s *SpannerScanner) FetchEntriesSince(ctx context.Context, since time.Time, limit int) ([]Entry, error) {
	const baseSQL = `
        SELECT UserId, MemoryId, CreationTime, EntryId, RawEntry, Summary, Metadata, Tags
        FROM MemoryEntries
        WHERE CreationTime > @since AND DeletionScheduledTime IS NULL
        ORDER BY CreationTime ASC
        LIMIT @limit`

	stmt := spanner.Statement{
		SQL: baseSQL,
		Params: map[string]interface{}{
			"since": since,
			"limit": limit,
		},
	}

	itr := s.client.Single().Query(ctx, stmt)
	defer itr.Stop()

	var res []Entry
	for {
		row, err := itr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var e Entry
		var metaJSON, tagsJSON spanner.NullJSON
		if err := row.Columns(&e.UserID, &e.MemoryID, &e.CreationTime, &e.EntryID, &e.RawEntry, &e.Summary, &metaJSON, &tagsJSON); err != nil {
			return nil, err
		}
		if metaJSON.Valid {
			e.Metadata = metaJSON.Value.(map[string]interface{})
		}
		if tagsJSON.Valid {
			e.Tags = tagsJSON.Value.(map[string]interface{})
		}
		res = append(res, e)
	}
	return res, nil
}

// FetchContextsSince returns context snapshots created strictly after `since` (ascending order).
func (s *SpannerScanner) FetchContextsSince(ctx context.Context, since time.Time, limit int) ([]ContextSnapshot, error) {
	const sql = `SELECT UserId, MemoryId, ContextId, CreationTime, Context
                 FROM MemoryContexts
                 WHERE CreationTime > @since
                 ORDER BY CreationTime ASC
                 LIMIT @limit`

	stmt := spanner.Statement{
		SQL: sql,
		Params: map[string]interface{}{
			"since": since,
			"limit": limit,
		},
	}

	itr := s.client.Single().Query(ctx, stmt)
	defer itr.Stop()

	var out []ContextSnapshot
	for {
		row, err := itr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var snap ContextSnapshot
		var ctxJSON spanner.NullJSON
		if err := row.Columns(&snap.UserID, &snap.MemoryID, &snap.ContextID, &snap.CreationTime, &ctxJSON); err != nil {
			return nil, err
		}
		// stringify JSON (minified) if valid
		if ctxJSON.Valid {
			if b, err := json.Marshal(ctxJSON.Value); err == nil {
				snap.Text = string(b)
			}
		}
		out = append(out, snap)
	}
	return out, nil
}
