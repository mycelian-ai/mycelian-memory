package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/store"
)

// Open opens a PostgreSQL connection using the pgx stdlib driver and verifies connectivity.
func Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres DSN is empty")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// NewWithDB constructs a native Postgres store backed directly by database/sql.
func NewWithDB(db *sql.DB) store.Store { return &pgStore{db: db} }

type pgStore struct{ db *sql.DB }

func (s *pgStore) Users() store.Users       { return &users{db: s.db} }
func (s *pgStore) Vaults() store.Vaults     { return &vaults{db: s.db} }
func (s *pgStore) Memories() store.Memories { return &memories{db: s.db} }
func (s *pgStore) Entries() store.Entries   { return &entries{db: s.db} }
func (s *pgStore) Contexts() store.Contexts { return &contexts{db: s.db} }

// HealthPing implements health.HealthPinger for Postgres-backed store.
func (s *pgStore) HealthPing(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Bootstrap performs a connectivity check to ensure Postgres is reachable.
// This is a fast ping-only check since compose migrations handle schema setup.
func Bootstrap(ctx context.Context, dsn string) error {
	if dsn == "" {
		return nil // No DSN configured, skip bootstrap
	}

	db, err := Open(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	return db.PingContext(ctx)
}

// --- Users ---
type users struct{ db *sql.DB }

func (u *users) Create(ctx context.Context, m *model.User) (*model.User, error) {
	var created time.Time
	row := u.db.QueryRowContext(ctx, `
        INSERT INTO users (user_id, email, display_name, time_zone, status)
        VALUES ($1,$2,$3,$4,'ACTIVE')
        RETURNING creation_time
    `, m.UserID, m.Email, m.DisplayName, m.TimeZone)
	if err := row.Scan(&created); err != nil {
		return nil, err
	}
	out := *m
	out.Status = "ACTIVE"
	out.CreationTime = created
	return &out, nil
}

func (u *users) Get(ctx context.Context, userID string) (*model.User, error) {
	var out model.User
	var last *time.Time
	row := u.db.QueryRowContext(ctx, `
        SELECT user_id, email, display_name, time_zone, status, creation_time, last_active_time
        FROM users WHERE user_id=$1
    `, userID)
	if err := row.Scan(&out.UserID, &out.Email, &out.DisplayName, &out.TimeZone, &out.Status, &out.CreationTime, &last); err != nil {
		return nil, err
	}
	out.LastActiveTime = last
	return &out, nil
}

func (u *users) Delete(ctx context.Context, userID string) error {
	// Not supported yet (no cascade in schema). Return not implemented.
	return errors.New("users.Delete not implemented")
}

// --- Vaults ---
type vaults struct{ db *sql.DB }

func (v *vaults) Create(ctx context.Context, mv *model.Vault) (*model.Vault, error) {
	id := mv.VaultID
	if id == "" {
		id = uuid.New().String()
	}
	var created time.Time
	row := v.db.QueryRowContext(ctx, `
        INSERT INTO vaults (actor_id, vault_id, title, description)
        VALUES ($1,$2,$3,$4)
        RETURNING creation_time
    `, mv.ActorID, id, mv.Title, nil)
	if err := row.Scan(&created); err != nil {
		return nil, err
	}
	return &model.Vault{VaultID: id, ActorID: mv.ActorID, Title: mv.Title, CreationTime: created}, nil
}

func (v *vaults) GetByID(ctx context.Context, userID, vaultID string) (*model.Vault, error) {
	var out model.Vault
	out.ActorID = userID
	out.VaultID = vaultID
	row := v.db.QueryRowContext(ctx, `
        SELECT title, description, creation_time FROM vaults WHERE user_id=$1 AND vault_id=$2
    `, userID, vaultID)
	var created time.Time
	var desc *string
	if err := row.Scan(&out.Title, &desc, &created); err != nil {
		return nil, err
	}
	out.CreationTime = created
	return &out, nil
}

func (v *vaults) GetByTitle(ctx context.Context, userID, title string) (*model.Vault, error) {
	var out model.Vault
	out.ActorID = userID
	out.Title = title
	row := v.db.QueryRowContext(ctx, `
        SELECT vault_id, description, creation_time FROM vaults WHERE actor_id=$1 AND title=$2
    `, userID, title)
	var created time.Time
	var desc *string
	if err := row.Scan(&out.VaultID, &desc, &created); err != nil {
		return nil, err
	}
	out.CreationTime = created
	return &out, nil
}

func (v *vaults) List(ctx context.Context, userID string) ([]*model.Vault, error) {
	rows, err := v.db.QueryContext(ctx, `
        SELECT vault_id, title, description, creation_time
        FROM vaults WHERE actor_id=$1 ORDER BY creation_time DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var res []*model.Vault
	for rows.Next() {
		var id, title string
		var desc *string
		var created time.Time
		if err := rows.Scan(&id, &title, &desc, &created); err != nil {
			return nil, err
		}
		res = append(res, &model.Vault{VaultID: id, ActorID: userID, Title: title, CreationTime: created})
	}
	return res, rows.Err()
}

func (v *vaults) Delete(ctx context.Context, userID, vaultID string) error {
	tx, err := v.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Collect child IDs
	entryRows, err := tx.QueryContext(ctx, `SELECT entry_id FROM memory_entries WHERE actor_id=$1 AND vault_id=$2`, userID, vaultID)
	if err != nil {
		return err
	}
	var entryIDs []string
	for entryRows.Next() {
		var id string
		if err := entryRows.Scan(&id); err != nil {
			_ = entryRows.Close()
			return err
		}
		entryIDs = append(entryIDs, id)
	}
	_ = entryRows.Close()

	ctxRows, err := tx.QueryContext(ctx, `SELECT context_id FROM memory_contexts WHERE actor_id=$1 AND vault_id=$2`, userID, vaultID)
	if err != nil {
		return err
	}
	var ctxIDs []string
	for ctxRows.Next() {
		var id string
		if err := ctxRows.Scan(&id); err != nil {
			_ = ctxRows.Close()
			return err
		}
		ctxIDs = append(ctxIDs, id)
	}
	_ = ctxRows.Close()

	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_entries WHERE actor_id=$1 AND vault_id=$2`, userID, vaultID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_contexts WHERE actor_id=$1 AND vault_id=$2`, userID, vaultID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memories WHERE actor_id=$1 AND vault_id=$2`, userID, vaultID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM vaults WHERE actor_id=$1 AND vault_id=$2`, userID, vaultID); err != nil {
		return err
	}

	for _, id := range entryIDs {
		if err := writeOutbox(ctx, tx, "delete_entry", id, map[string]interface{}{"actorId": userID}); err != nil {
			return err
		}
	}
	for _, id := range ctxIDs {
		if err := writeOutbox(ctx, tx, "delete_context", id, map[string]interface{}{"actorId": userID}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (v *vaults) AddMemory(ctx context.Context, userID, vaultID, memoryID string) error {
	tx, err := v.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Validate target vault exists
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM vaults WHERE actor_id=$1 AND vault_id=$2`, userID, vaultID).Scan(&exists); err != nil {
		return fmt.Errorf("VAULT_NOT_FOUND: %w", err)
	}

	// Locate current vault and title for the memory
	var currentVaultID, title string
	if err := tx.QueryRowContext(ctx, `SELECT vault_id, title FROM memories WHERE actor_id=$1 AND memory_id=$2`, userID, memoryID).Scan(&currentVaultID, &title); err != nil {
		return fmt.Errorf("MEMORY_NOT_FOUND: %w", err)
	}
	if currentVaultID == vaultID {
		return tx.Commit()
	}

	// Enforce unique (vault_id, title) in target
	var conflict int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM memories WHERE actor_id=$1 AND vault_id=$2 AND title=$3`, userID, vaultID, title).Scan(&conflict)
	if err == nil {
		return fmt.Errorf("MEMORY_TITLE_CONFLICT: title already exists in target vault")
	}
	if err != sql.ErrNoRows {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE memories SET vault_id=$1 WHERE actor_id=$2 AND vault_id=$3 AND memory_id=$4`, vaultID, userID, currentVaultID, memoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_entries SET vault_id=$1 WHERE actor_id=$2 AND vault_id=$3 AND memory_id=$4`, vaultID, userID, currentVaultID, memoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_contexts SET vault_id=$1 WHERE actor_id=$2 AND vault_id=$3 AND memory_id=$4`, vaultID, userID, currentVaultID, memoryID); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Memories ---
type memories struct{ db *sql.DB }

func (m *memories) Create(ctx context.Context, mm *model.Memory) (*model.Memory, error) {
	tx, err := m.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	memID := uuid.New().String()
	var created time.Time
	if err := tx.QueryRowContext(ctx, `
        INSERT INTO memories (actor_id, vault_id, memory_id, memory_type, title, description)
        VALUES ($1,$2,$3,$4,$5,$6)
        RETURNING creation_time
    `, mm.ActorID, mm.VaultID, memID, mm.MemoryType, mm.Title, mm.Description).Scan(&created); err != nil {
		return nil, err
	}

	// default context snapshot
	ctxID := uuid.New().String()
	defaultCtx := json.RawMessage(`{"activeContext":"This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available."}`)
	var ctxCreated time.Time
	if err := tx.QueryRowContext(ctx, `
        INSERT INTO memory_contexts (actor_id, vault_id, memory_id, context_id, context)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING creation_time
    `, mm.ActorID, mm.VaultID, memID, ctxID, []byte(defaultCtx)).Scan(&ctxCreated); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"actorId":      mm.ActorID,
		"memoryId":     memID,
		"contextId":    ctxID,
		"context":      string(defaultCtx),
		"creationTime": ctxCreated,
	}
	if err := writeOutbox(ctx, tx, "upsert_context", ctxID, payload); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &model.Memory{MemoryID: memID, ActorID: mm.ActorID, VaultID: mm.VaultID, MemoryType: mm.MemoryType, Title: mm.Title, Description: mm.Description, CreationTime: created}, nil
}

func (m *memories) GetByID(ctx context.Context, userID, vaultID, memoryID string) (*model.Memory, error) {
	var out model.Memory
	out.ActorID = userID
	out.VaultID = vaultID
	out.MemoryID = memoryID
	row := m.db.QueryRowContext(ctx, `
        SELECT memory_type, title, description, creation_time
        FROM memories WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3
    `, userID, vaultID, memoryID)
	if err := row.Scan(&out.MemoryType, &out.Title, &out.Description, &out.CreationTime); err != nil {
		return nil, err
	}
	return &out, nil
}

func (m *memories) GetByTitle(ctx context.Context, userID, vaultID, title string) (*model.Memory, error) {
	var out model.Memory
	out.ActorID = userID
	out.VaultID = vaultID
	out.Title = title
	row := m.db.QueryRowContext(ctx, `
        SELECT memory_id, memory_type, description, creation_time
        FROM memories WHERE actor_id=$1 AND vault_id=$2 AND title=$3
    `, userID, vaultID, title)
	if err := row.Scan(&out.MemoryID, &out.MemoryType, &out.Description, &out.CreationTime); err != nil {
		return nil, err
	}
	return &out, nil
}

func (m *memories) List(ctx context.Context, userID, vaultID string) ([]*model.Memory, error) {
	rows, err := m.db.QueryContext(ctx, `
        SELECT memory_id, memory_type, title, description, creation_time
        FROM memories WHERE actor_id=$1 AND vault_id=$2 ORDER BY creation_time DESC
    `, userID, vaultID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*model.Memory
	for rows.Next() {
		var mm model.Memory
		mm.ActorID = userID
		mm.VaultID = vaultID
		if err := rows.Scan(&mm.MemoryID, &mm.MemoryType, &mm.Title, &mm.Description, &mm.CreationTime); err != nil {
			return nil, err
		}
		out = append(out, &mm)
	}
	return out, rows.Err()
}

func (m *memories) Delete(ctx context.Context, userID, vaultID, memoryID string) error {
	tx, err := m.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	entryRows, err := tx.QueryContext(ctx, `SELECT entry_id FROM memory_entries WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID, memoryID)
	if err != nil {
		return err
	}
	var entryIDs []string
	for entryRows.Next() {
		var id string
		if err := entryRows.Scan(&id); err != nil {
			_ = entryRows.Close()
			return err
		}
		entryIDs = append(entryIDs, id)
	}
	_ = entryRows.Close()

	ctxRows, err := tx.QueryContext(ctx, `SELECT context_id FROM memory_contexts WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID, memoryID)
	if err != nil {
		return err
	}
	var ctxIDs []string
	for ctxRows.Next() {
		var id string
		if err := ctxRows.Scan(&id); err != nil {
			_ = ctxRows.Close()
			return err
		}
		ctxIDs = append(ctxIDs, id)
	}
	_ = ctxRows.Close()

	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_entries WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID, memoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_contexts WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID, memoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memories WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID, memoryID); err != nil {
		return err
	}

	for _, id := range entryIDs {
		if err := writeOutbox(ctx, tx, "delete_entry", id, map[string]interface{}{"actorId": userID}); err != nil {
			return err
		}
	}
	for _, id := range ctxIDs {
		if err := writeOutbox(ctx, tx, "delete_context", id, map[string]interface{}{"actorId": userID}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- Entries ---
type entries struct{ db *sql.DB }

func (e *entries) Create(ctx context.Context, me *model.MemoryEntry) (*model.MemoryEntry, error) {
	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	entryID := uuid.New().String()
	var created time.Time
	metaJSON, _ := json.Marshal(me.Metadata)
	tagsJSON, _ := json.Marshal(me.Tags)
	row := tx.QueryRowContext(ctx, `
        INSERT INTO memory_entries (actor_id, vault_id, memory_id, raw_entry, summary, metadata, tags, entry_id)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING creation_time
    `, me.ActorID, me.VaultID, me.MemoryID, me.RawEntry, me.Summary, nullIfEmpty(metaJSON), nullIfEmpty(tagsJSON), entryID)
	if err := row.Scan(&created); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"actorId":      me.ActorID,
		"memoryId":     me.MemoryID,
		"entryId":      entryID,
		"rawEntry":     me.RawEntry,
		"summary":      me.Summary,
		"tags":         me.Tags,
		"creationTime": created,
	}
	if err := writeOutbox(ctx, tx, "upsert_entry", entryID, payload); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	out := *me
	out.EntryID = entryID
	out.CreationTime = created
	return &out, nil
}

func (e *entries) List(ctx context.Context, req model.ListEntriesRequest) ([]*model.MemoryEntry, error) {
	query := `SELECT actor_id, vault_id, memory_id, creation_time, entry_id, raw_entry, summary, metadata, tags,
                      correction_time, corrected_entry_memory_id, corrected_entry_creation_time,
                      correction_reason, last_update_time
               FROM memory_entries WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3`
	args := []interface{}{req.ActorID, req.VaultID, req.MemoryID}
	if req.Before != nil {
		query += " AND creation_time < $4"
		args = append(args, *req.Before)
	}
	if req.After != nil && req.Before == nil {
		query += " AND creation_time > $4"
		args = append(args, *req.After)
	}
	query += " ORDER BY creation_time DESC"
	if req.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", req.Limit)
	}
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*model.MemoryEntry
	for rows.Next() {
		var m model.MemoryEntry
		var meta, tags sql.NullString
		var corrTime, corrEntryTime, lastUpd sql.NullTime
		var corrMemID sql.NullString
		if err := rows.Scan(&m.ActorID, &m.VaultID, &m.MemoryID, &m.CreationTime, &m.EntryID, &m.RawEntry, &m.Summary, &meta, &tags,
			&corrTime, &corrMemID, &corrEntryTime, &corrMemID, &lastUpd); err != nil {
			return nil, err
		}
		if meta.Valid {
			_ = json.Unmarshal([]byte(meta.String), &m.Metadata)
		}
		if tags.Valid {
			_ = json.Unmarshal([]byte(tags.String), &m.Tags)
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (e *entries) GetByID(ctx context.Context, userID, vaultID, memoryID, entryID string) (*model.MemoryEntry, error) {
	var m model.MemoryEntry
	var meta, tags sql.NullString
	var corrTime, corrEntryTime, lastUpd sql.NullTime
	var corrMemID sql.NullString
	row := e.db.QueryRowContext(ctx, `
        SELECT actor_id, vault_id, memory_id, creation_time, entry_id, raw_entry, summary, metadata, tags,
               correction_time, corrected_entry_memory_id, corrected_entry_creation_time,
               correction_reason, last_update_time
        FROM memory_entries WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3 AND entry_id=$4
    `, userID, vaultID, memoryID, entryID)
	if err := row.Scan(&m.ActorID, &m.VaultID, &m.MemoryID, &m.CreationTime, &m.EntryID, &m.RawEntry, &m.Summary, &meta, &tags,
		&corrTime, &corrMemID, &corrEntryTime, &corrMemID, &lastUpd); err != nil {
		return nil, err
	}
	if meta.Valid {
		_ = json.Unmarshal([]byte(meta.String), &m.Metadata)
	}
	if tags.Valid {
		_ = json.Unmarshal([]byte(tags.String), &m.Tags)
	}
	return &m, nil
}

func (e *entries) UpdateTags(ctx context.Context, userID, vaultID, memoryID, entryID string, tags map[string]interface{}) (*model.MemoryEntry, error) {
	tagsJSON, _ := json.Marshal(tags)
	if _, err := e.db.ExecContext(ctx, `UPDATE memory_entries SET tags=$1, last_update_time=now() WHERE actor_id=$2 AND vault_id=$3 AND memory_id=$4 AND entry_id=$5`, nullIfEmpty(tagsJSON), userID, vaultID, memoryID, entryID); err != nil {
		return nil, err
	}
	return e.GetByID(ctx, userID, vaultID, memoryID, entryID)
}

func (e *entries) DeleteByID(ctx context.Context, userID, vaultID, memoryID, entryID string) error {
	tx, err := e.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `DELETE FROM memory_entries WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3 AND entry_id=$4`, userID, vaultID, memoryID, entryID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if err := writeOutbox(ctx, tx, "delete_entry", entryID, map[string]interface{}{"actorId": userID}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- Contexts ---
type contexts struct{ db *sql.DB }

func (c *contexts) Put(ctx context.Context, mc *model.MemoryContext) (*model.MemoryContext, error) {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	ctxID := mc.ContextID
	if ctxID == "" {
		ctxID = uuid.New().String()
	}
	var created time.Time
	row := tx.QueryRowContext(ctx, `
        INSERT INTO memory_contexts (actor_id, vault_id, memory_id, context_id, context)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING creation_time
    `, mc.ActorID, mc.VaultID, mc.MemoryID, ctxID, []byte(mc.ContextJSON))
	if err := row.Scan(&created); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"actorId":      mc.ActorID,
		"memoryId":     mc.MemoryID,
		"contextId":    ctxID,
		"context":      string(mc.ContextJSON),
		"creationTime": created,
	}
	if err := writeOutbox(ctx, tx, "upsert_context", ctxID, payload); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	out := *mc
	out.ContextID = ctxID
	out.CreationTime = created
	return &out, nil
}

func (c *contexts) Latest(ctx context.Context, userID, vaultID, memoryID string) (*model.MemoryContext, error) {
	var out model.MemoryContext
	out.ActorID = userID
	out.VaultID = vaultID
	out.MemoryID = memoryID
	var ctxBytes []byte
	row := c.db.QueryRowContext(ctx, `
        SELECT context_id, context, creation_time
        FROM memory_contexts WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3
        ORDER BY creation_time DESC LIMIT 1
    `, userID, vaultID, memoryID)
	if err := row.Scan(&out.ContextID, &ctxBytes, &out.CreationTime); err != nil {
		return nil, err
	}
	out.ContextJSON = ctxBytes
	return &out, nil
}

func (c *contexts) DeleteByID(ctx context.Context, userID, vaultID, memoryID, contextID string) error {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `DELETE FROM memory_contexts WHERE actor_id=$1 AND vault_id=$2 AND memory_id=$3 AND context_id=$4`, userID, vaultID, memoryID, contextID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if err := writeOutbox(ctx, tx, "delete_context", contextID, map[string]interface{}{"actorId": userID}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// helpers
func writeOutbox(ctx context.Context, tx *sql.Tx, op string, aggregateID string, payload map[string]interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO outbox (aggregate_id, op, payload) VALUES ($1,$2,$3)`, aggregateID, op, b)
	return err
}

func nullIfEmpty(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
