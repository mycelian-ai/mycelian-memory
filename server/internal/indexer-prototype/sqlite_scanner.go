package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/mycelian/mycelian-memory/server/internal/storage/sqlite"

	"github.com/rs/zerolog"
)

// sqliteScanner implements Scanner for the local SQLite backend used in dev/E2E.
// It shares the same database file as the memory-service container (path provided
// via Config.SQLitePath).

type sqliteScanner struct {
	db  *sql.DB
	log zerolog.Logger
}

// NewSQLiteScanner opens the SQLite database in read-only mode (but WAL still
// works cross-process). It does not create schema; assumes memory-service has
// done that already.
func NewSQLiteScanner(path string, log zerolog.Logger) (Scanner, error) {
	db, err := sqlite.Open(path)
	if err != nil {
		return nil, err
	}
	return &sqliteScanner{db: db, log: log.With().Str("component", "sqliteScanner").Logger()}, nil
}

func (s *sqliteScanner) Close() error {
	return s.db.Close()
}

func (s *sqliteScanner) FetchEntriesSince(ctx context.Context, since time.Time, limit int) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT UserId, MemoryId, CreationTime, EntryId, RawEntry, Summary, Metadata, Tags 
        FROM MemoryEntries WHERE CreationTime > ? ORDER BY CreationTime ASC LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var e Entry
		var metaStr, tagsStr sql.NullString
		if err := rows.Scan(&e.UserID, &e.MemoryID, &e.CreationTime, &e.EntryID, &e.RawEntry, &e.Summary, &metaStr, &tagsStr); err != nil {
			return nil, err
		}
		if metaStr.Valid {
			_ = json.Unmarshal([]byte(metaStr.String), &e.Metadata)
		}
		if tagsStr.Valid {
			_ = json.Unmarshal([]byte(tagsStr.String), &e.Tags)
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *sqliteScanner) FetchContextsSince(ctx context.Context, since time.Time, limit int) ([]ContextSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT UserId, MemoryId, ContextId, CreationTime, Context FROM MemoryContexts WHERE CreationTime > ? ORDER BY CreationTime ASC LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ContextSnapshot
	for rows.Next() {
		var snap ContextSnapshot
		var ctxJSON string
		if err := rows.Scan(&snap.UserID, &snap.MemoryID, &snap.ContextID, &snap.CreationTime, &ctxJSON); err != nil {
			return nil, err
		}
		snap.Text = ctxJSON
		list = append(list, snap)
	}
	return list, nil
}
