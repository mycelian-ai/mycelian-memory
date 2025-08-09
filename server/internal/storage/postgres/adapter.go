package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/google/uuid"

	"github.com/mycelian/mycelian-memory/server/internal/storage"
)

// PostgresStorage implements storage.Storage using PostgreSQL via database/sql (pgx driver).
type PostgresStorage struct {
	db *sql.DB
}

// ErrNotImplemented is returned by methods that are not yet implemented.
var ErrNotImplemented = errors.New("postgres adapter: not implemented")

// Open returns a *sql.DB using the pgx stdlib driver.
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

// NewPostgresStorageWithDB constructs a storage adapter from an existing DB connection.
func NewPostgresStorageWithDB(db *sql.DB) (storage.Storage, error) {
	if db == nil {
		return nil, fmt.Errorf("nil db")
	}
	return &PostgresStorage{db: db}, nil
}

// --- Health ---

func (s *PostgresStorage) HealthCheck(ctx context.Context) error {
	// Simple query to validate connectivity
	row := s.db.QueryRowContext(ctx, "SELECT 1")
	var one int
	return row.Scan(&one)
}

// --- User operations ---

func (s *PostgresStorage) CreateUser(ctx context.Context, req storage.CreateUserRequest) (*storage.User, error) {
	var u storage.User
	u.UserID = req.UserID
	u.Email = req.Email
	u.DisplayName = req.DisplayName
	u.TimeZone = req.TimeZone
	u.Status = "ACTIVE"

	// Use DEFAULT now() for creation_time; RETURNING to fetch
	row := s.db.QueryRowContext(ctx, `
        INSERT INTO users (user_id, email, display_name, time_zone, status)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING creation_time
    `, u.UserID, u.Email, u.DisplayName, u.TimeZone, u.Status)
	if err := row.Scan(&u.CreationTime); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStorage) GetUser(ctx context.Context, userID string) (*storage.User, error) {
	var u storage.User
	row := s.db.QueryRowContext(ctx, `
        SELECT user_id, email, display_name, time_zone, status, creation_time, last_active_time
        FROM users WHERE user_id=$1
    `, userID)
	if err := row.Scan(&u.UserID, &u.Email, &u.DisplayName, &u.TimeZone, &u.Status, &u.CreationTime, &u.LastActiveTime); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStorage) GetUserByEmail(ctx context.Context, email string) (*storage.User, error) {
	var u storage.User
	row := s.db.QueryRowContext(ctx, `
        SELECT user_id, email, display_name, time_zone, status, creation_time, last_active_time
        FROM users WHERE email=$1 AND status='ACTIVE'
    `, email)
	if err := row.Scan(&u.UserID, &u.Email, &u.DisplayName, &u.TimeZone, &u.Status, &u.CreationTime, &u.LastActiveTime); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStorage) UpdateUserLastActive(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET last_active_time = now() WHERE user_id=$1`, userID)
	return err
}

// --- Vault operations ---

func (s *PostgresStorage) CreateVault(ctx context.Context, req storage.CreateVaultRequest) (*storage.Vault, error) {
	var v storage.Vault
	v.UserID = req.UserID
	v.VaultID = req.VaultID
	v.Title = req.Title
	v.Description = req.Description
	row := s.db.QueryRowContext(ctx, `
        INSERT INTO vaults (user_id, vault_id, title, description)
        VALUES ($1,$2,$3,$4)
        RETURNING creation_time
    `, v.UserID, v.VaultID.String(), v.Title, v.Description)
	if err := row.Scan(&v.CreationTime); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *PostgresStorage) GetVault(ctx context.Context, userID string, vaultID uuid.UUID) (*storage.Vault, error) {
	var v storage.Vault
	v.UserID = userID
	v.VaultID = vaultID
	row := s.db.QueryRowContext(ctx, `
        SELECT title, description, creation_time FROM vaults WHERE user_id=$1 AND vault_id=$2
    `, userID, vaultID.String())
	if err := row.Scan(&v.Title, &v.Description, &v.CreationTime); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *PostgresStorage) ListVaults(ctx context.Context, userID string) ([]*storage.Vault, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT vault_id, title, description, creation_time
        FROM vaults WHERE user_id=$1 ORDER BY creation_time DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*storage.Vault
	for rows.Next() {
		var v storage.Vault
		v.UserID = userID
		if err := rows.Scan(&v.VaultID, &v.Title, &v.Description, &v.CreationTime); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) GetVaultByTitle(ctx context.Context, userID string, title string) (*storage.Vault, error) {
	var v storage.Vault
	v.UserID = userID
	v.Title = title
	row := s.db.QueryRowContext(ctx, `
        SELECT vault_id, description, creation_time FROM vaults WHERE user_id=$1 AND title=$2
    `, userID, title)
	if err := row.Scan(&v.VaultID, &v.Description, &v.CreationTime); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *PostgresStorage) DeleteVault(ctx context.Context, userID string, vaultID uuid.UUID) error {
	// Collect all child IDs across memories in this vault and enqueue deletes
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Entries and contexts scoped by user+vault
	entryRows, err := tx.QueryContext(ctx, `SELECT entry_id FROM memory_entries WHERE user_id=$1 AND vault_id=$2`, userID, vaultID.String())
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

	ctxRows, err := tx.QueryContext(ctx, `SELECT context_id FROM memory_contexts WHERE user_id=$1 AND vault_id=$2`, userID, vaultID.String())
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

	// Delete children and parent objects
	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_entries WHERE user_id=$1 AND vault_id=$2`, userID, vaultID.String()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_contexts WHERE user_id=$1 AND vault_id=$2`, userID, vaultID.String()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memories WHERE user_id=$1 AND vault_id=$2`, userID, vaultID.String()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM vaults WHERE user_id=$1 AND vault_id=$2`, userID, vaultID.String()); err != nil {
		return err
	}

	// Enqueue outbox deletes (idempotent)
	for _, id := range entryIDs {
		if err := writeOutbox(ctx, tx, "delete_entry", id, map[string]interface{}{"userId": userID}); err != nil {
			return err
		}
	}
	for _, id := range ctxIDs {
		if err := writeOutbox(ctx, tx, "delete_context", id, map[string]interface{}{"userId": userID}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// --- Memory operations ---

func (s *PostgresStorage) CreateMemory(ctx context.Context, req storage.CreateMemoryRequest) (*storage.Memory, error) {
	// Insert memory and default context snapshot, with outbox upsert_context, in one tx
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	memoryID := uuid.New().String()
	var mem storage.Memory
	mem.UserID = req.UserID
	mem.VaultID = req.VaultID
	mem.MemoryID = memoryID
	mem.MemoryType = req.MemoryType
	mem.Title = req.Title
	mem.Description = req.Description

	if err := tx.QueryRowContext(ctx, `
        INSERT INTO memories (user_id, vault_id, memory_id, memory_type, title, description)
        VALUES ($1,$2,$3,$4,$5,$6)
        RETURNING creation_time
    `, mem.UserID, mem.VaultID.String(), mem.MemoryID, mem.MemoryType, mem.Title, mem.Description).Scan(&mem.CreationTime); err != nil {
		return nil, err
	}

	// default context
	ctxID := uuid.New().String()
	defaultCtx := json.RawMessage(`{"activeContext":"This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available."}`)
	var ctxCreated time.Time
	if err := tx.QueryRowContext(ctx, `
        INSERT INTO memory_contexts (user_id, vault_id, memory_id, context_id, context)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING creation_time
    `, mem.UserID, mem.VaultID.String(), mem.MemoryID, ctxID, []byte(defaultCtx)).Scan(&ctxCreated); err != nil {
		return nil, err
	}

	// outbox upsert_context
	payload := map[string]interface{}{
		"userId":       mem.UserID,
		"memoryId":     mem.MemoryID,
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
	return &mem, nil
}

func (s *PostgresStorage) GetMemory(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) (*storage.Memory, error) {
	var m storage.Memory
	m.UserID = userID
	m.VaultID = vaultID
	m.MemoryID = memoryID
	row := s.db.QueryRowContext(ctx, `
        SELECT memory_type, title, description, creation_time
        FROM memories WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3
    `, userID, vaultID.String(), memoryID)
	if err := row.Scan(&m.MemoryType, &m.Title, &m.Description, &m.CreationTime); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *PostgresStorage) ListMemories(ctx context.Context, userID string, vaultID uuid.UUID) ([]*storage.Memory, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT memory_id, memory_type, title, description, creation_time
        FROM memories WHERE user_id=$1 AND vault_id=$2 ORDER BY creation_time DESC
    `, userID, vaultID.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*storage.Memory
	for rows.Next() {
		var m storage.Memory
		m.UserID = userID
		m.VaultID = vaultID
		if err := rows.Scan(&m.MemoryID, &m.MemoryType, &m.Title, &m.Description, &m.CreationTime); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) GetMemoryByTitle(ctx context.Context, userID string, vaultID uuid.UUID, title string) (*storage.Memory, error) {
	var m storage.Memory
	m.UserID = userID
	m.VaultID = vaultID
	m.Title = title
	row := s.db.QueryRowContext(ctx, `
        SELECT memory_id, memory_type, description, creation_time
        FROM memories WHERE user_id=$1 AND vault_id=$2 AND title=$3
    `, userID, vaultID.String(), title)
	if err := row.Scan(&m.MemoryID, &m.MemoryType, &m.Description, &m.CreationTime); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *PostgresStorage) DeleteMemory(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) error {
	// Ensure child rows are removed and enqueue outbox deletes for search cleanup
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Collect child IDs before deletion
	entryRows, err := tx.QueryContext(ctx, `SELECT entry_id FROM memory_entries WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID.String(), memoryID)
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

	ctxRows, err := tx.QueryContext(ctx, `SELECT context_id FROM memory_contexts WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID.String(), memoryID)
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

	// Delete children and parent
	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_entries WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID.String(), memoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_contexts WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID.String(), memoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM memories WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3`, userID, vaultID.String(), memoryID); err != nil {
		return err
	}

	// Enqueue outbox deletes (idempotent)
	for _, id := range entryIDs {
		if err := writeOutbox(ctx, tx, "delete_entry", id, map[string]interface{}{"userId": userID}); err != nil {
			return err
		}
	}
	for _, id := range ctxIDs {
		if err := writeOutbox(ctx, tx, "delete_context", id, map[string]interface{}{"userId": userID}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// --- Memory entry operations ---

func (s *PostgresStorage) CreateMemoryEntry(ctx context.Context, req storage.CreateMemoryEntryRequest) (*storage.MemoryEntry, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	entryID := uuid.New().String()
	var creation time.Time
	metaJSON, _ := json.Marshal(req.Metadata)
	tagsJSON, _ := json.Marshal(req.Tags)
	row := tx.QueryRowContext(ctx, `
        INSERT INTO memory_entries (user_id, vault_id, memory_id, raw_entry, summary, metadata, tags, entry_id)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING creation_time
    `, req.UserID, req.VaultID.String(), req.MemoryID, req.RawEntry, req.Summary, nullIfEmpty(metaJSON), nullIfEmpty(tagsJSON), entryID)
	if err := row.Scan(&creation); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"userId":       req.UserID,
		"memoryId":     req.MemoryID,
		"entryId":      entryID,
		"rawEntry":     req.RawEntry,
		"summary":      req.Summary,
		"creationTime": creation,
	}
	if err := writeOutbox(ctx, tx, "upsert_entry", entryID, payload); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &storage.MemoryEntry{
		UserID:       req.UserID,
		VaultID:      req.VaultID,
		MemoryID:     req.MemoryID,
		CreationTime: creation,
		EntryID:      entryID,
		RawEntry:     req.RawEntry,
		Summary:      req.Summary,
		Metadata:     req.Metadata,
		Tags:         req.Tags,
	}, nil
}

func (s *PostgresStorage) GetMemoryEntry(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, creationTime time.Time) (*storage.MemoryEntry, error) {
	var e storage.MemoryEntry
	e.UserID = userID
	e.VaultID = vaultID
	e.MemoryID = memoryID
	e.CreationTime = creationTime
	var meta, tags sql.NullString
	var corrTime, corrEntryTime, lastUpd sql.NullTime
	var corrMemID, corrReason sql.NullString
	row := s.db.QueryRowContext(ctx, `
        SELECT entry_id, raw_entry, summary, metadata, tags,
               correction_time, corrected_entry_memory_id, corrected_entry_creation_time,
               correction_reason, last_update_time
        FROM memory_entries WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3 AND creation_time=$4
    `, userID, vaultID.String(), memoryID, creationTime)
	if err := row.Scan(&e.EntryID, &e.RawEntry, &e.Summary, &meta, &tags,
		&corrTime, &corrMemID, &corrEntryTime, &corrReason, &lastUpd); err != nil {
		return nil, err
	}
	if meta.Valid {
		_ = json.Unmarshal([]byte(meta.String), &e.Metadata)
	}
	if tags.Valid {
		_ = json.Unmarshal([]byte(tags.String), &e.Tags)
	}
	if corrTime.Valid {
		t := corrTime.Time
		e.CorrectionTime = &t
	}
	if corrMemID.Valid {
		s := corrMemID.String
		e.CorrectedEntryMemoryId = &s
	}
	if corrEntryTime.Valid {
		t := corrEntryTime.Time
		e.CorrectedEntryCreationTime = &t
	}
	if corrReason.Valid {
		s := corrReason.String
		e.CorrectionReason = &s
	}
	if lastUpd.Valid {
		t := lastUpd.Time
		e.LastUpdateTime = &t
	}
	return &e, nil
}

func (s *PostgresStorage) GetMemoryEntryByID(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, entryID string) (*storage.MemoryEntry, error) {
	var e storage.MemoryEntry
	var vid string
	var meta, tags sql.NullString
	var corrTime, corrEntryTime, lastUpd sql.NullTime
	var corrMemID, corrReason sql.NullString
	row := s.db.QueryRowContext(ctx, `
        SELECT user_id, vault_id, memory_id, creation_time, entry_id, raw_entry, summary, metadata, tags,
               correction_time, corrected_entry_memory_id, corrected_entry_creation_time,
               correction_reason, last_update_time
        FROM memory_entries WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3 AND entry_id=$4
    `, userID, vaultID.String(), memoryID, entryID)
	if err := row.Scan(&e.UserID, &vid, &e.MemoryID, &e.CreationTime, &e.EntryID, &e.RawEntry, &e.Summary, &meta, &tags,
		&corrTime, &corrMemID, &corrEntryTime, &corrReason, &lastUpd); err != nil {
		return nil, err
	}
	if v, err := uuid.Parse(vid); err == nil {
		e.VaultID = v
	}
	if meta.Valid {
		_ = json.Unmarshal([]byte(meta.String), &e.Metadata)
	}
	if tags.Valid {
		_ = json.Unmarshal([]byte(tags.String), &e.Tags)
	}
	if corrTime.Valid {
		t := corrTime.Time
		e.CorrectionTime = &t
	}
	if corrMemID.Valid {
		s := corrMemID.String
		e.CorrectedEntryMemoryId = &s
	}
	if corrEntryTime.Valid {
		t := corrEntryTime.Time
		e.CorrectedEntryCreationTime = &t
	}
	if corrReason.Valid {
		s := corrReason.String
		e.CorrectionReason = &s
	}
	if lastUpd.Valid {
		t := lastUpd.Time
		e.LastUpdateTime = &t
	}
	return &e, nil
}

func (s *PostgresStorage) ListMemoryEntries(ctx context.Context, req storage.ListMemoryEntriesRequest) ([]*storage.MemoryEntry, error) {
	query := `SELECT user_id, vault_id, memory_id, creation_time, entry_id, raw_entry, summary, metadata, tags,
                     correction_time, corrected_entry_memory_id, corrected_entry_creation_time,
                     correction_reason, last_update_time
              FROM memory_entries WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3`
	args := []interface{}{req.UserID, req.VaultID.String(), req.MemoryID}
	if req.Before != nil {
		query += " AND creation_time < $4"
		args = append(args, *req.Before)
	}
	// If both After and Before set, parameter positions shift; for simplicity, handle mutually exclusive filters here
	if req.After != nil && req.Before == nil {
		query += " AND creation_time > $4"
		args = append(args, *req.After)
	}
	query += " ORDER BY creation_time DESC"
	if req.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", req.Limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*storage.MemoryEntry
	for rows.Next() {
		var e storage.MemoryEntry
		var vid string
		var meta, tags sql.NullString
		var corrTime, corrEntryTime, lastUpd sql.NullTime
		var corrMemID, corrReason sql.NullString
		if err := rows.Scan(&e.UserID, &vid, &e.MemoryID, &e.CreationTime, &e.EntryID, &e.RawEntry, &e.Summary, &meta, &tags,
			&corrTime, &corrMemID, &corrEntryTime, &corrReason, &lastUpd); err != nil {
			return nil, err
		}
		if v, err := uuid.Parse(vid); err == nil {
			e.VaultID = v
		}
		if meta.Valid {
			_ = json.Unmarshal([]byte(meta.String), &e.Metadata)
		}
		if tags.Valid {
			_ = json.Unmarshal([]byte(tags.String), &e.Tags)
		}
		if corrTime.Valid {
			t := corrTime.Time
			e.CorrectionTime = &t
		}
		if corrMemID.Valid {
			s := corrMemID.String
			e.CorrectedEntryMemoryId = &s
		}
		if corrEntryTime.Valid {
			t := corrEntryTime.Time
			e.CorrectedEntryCreationTime = &t
		}
		if corrReason.Valid {
			s := corrReason.String
			e.CorrectionReason = &s
		}
		if lastUpd.Valid {
			t := lastUpd.Time
			e.LastUpdateTime = &t
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (s *PostgresStorage) CorrectMemoryEntry(ctx context.Context, req storage.CorrectMemoryEntryRequest) (*storage.MemoryEntry, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// Verify original entry exists and is correctable (scoped by vault)
	var existingCorrection *time.Time
	row := tx.QueryRowContext(ctx, `
        SELECT correction_time FROM memory_entries
        WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3 AND creation_time=$4
        LIMIT 1
    `, req.UserID, req.VaultID.String(), req.MemoryID, req.OriginalCreationTime)
	if err := row.Scan(&existingCorrection); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ENTRY_NOT_FOUND: entry does not exist")
		}
		return nil, fmt.Errorf("failed to check original entry: %w", err)
	}
	if existingCorrection != nil {
		return nil, fmt.Errorf("IMMUTABILITY_VIOLATION: entry was already corrected at %v", *existingCorrection)
	}

	// Insert correction entry
	var created time.Time
	metaJSON, _ := json.Marshal(req.Metadata)
	tagsJSON, _ := json.Marshal(req.Tags)
	if err := tx.QueryRowContext(ctx, `
        INSERT INTO memory_entries (user_id, vault_id, memory_id, raw_entry, summary, metadata, tags, entry_id)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING creation_time
    `, req.UserID, req.VaultID.String(), req.MemoryID, req.CorrectedContent, req.CorrectedSummary, nullIfEmpty(metaJSON), nullIfEmpty(tagsJSON), req.CorrectedEntryID).Scan(&created); err != nil {
		return nil, fmt.Errorf("failed to insert correction entry: %w", err)
	}

	// Mark original as corrected
	if _, err := tx.ExecContext(ctx, `
        UPDATE memory_entries
        SET correction_time = now(),
            corrected_entry_memory_id = $1,
            corrected_entry_creation_time = $2,
            correction_reason = $3
        WHERE user_id=$4 AND vault_id=$5 AND memory_id=$6 AND creation_time=$7
    `, req.MemoryID, created, req.CorrectionReason, req.UserID, req.VaultID.String(), req.MemoryID, req.OriginalCreationTime); err != nil {
		return nil, fmt.Errorf("failed to update original entry: %w", err)
	}

	// Outbox for correction entry upsert
	payload := map[string]interface{}{
		"userId":       req.UserID,
		"memoryId":     req.MemoryID,
		"entryId":      req.CorrectedEntryID,
		"rawEntry":     req.CorrectedContent,
		"summary":      req.CorrectedSummary,
		"creationTime": created,
	}
	if err := writeOutbox(ctx, tx, "upsert_entry", req.CorrectedEntryID, payload); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &storage.MemoryEntry{
		UserID:       req.UserID,
		MemoryID:     req.MemoryID,
		CreationTime: created,
		EntryID:      req.CorrectedEntryID,
		RawEntry:     req.CorrectedContent,
		Summary:      req.CorrectedSummary,
		Metadata:     req.Metadata,
		Tags:         req.Tags,
	}, nil
}

func (s *PostgresStorage) UpdateMemoryEntrySummary(ctx context.Context, req storage.UpdateMemoryEntrySummaryRequest) (*storage.MemoryEntry, error) {
	// Simple scoped update: ensure vault isolation
	_, err := s.db.ExecContext(ctx, `UPDATE memory_entries SET summary=$1, last_update_time=now() WHERE user_id=$2 AND vault_id=$3 AND memory_id=$4 AND entry_id=$5`, req.Summary, req.UserID, req.VaultID.String(), req.MemoryID, req.EntryID)
	if err != nil {
		return nil, err
	}
	return s.GetMemoryEntryByID(ctx, req.UserID, req.VaultID, req.MemoryID, req.EntryID)
}

func (s *PostgresStorage) UpdateMemoryEntryTags(ctx context.Context, req storage.UpdateMemoryEntryTagsRequest) (*storage.MemoryEntry, error) {
	tagsJSON, _ := json.Marshal(req.Tags)
	_, err := s.db.ExecContext(ctx, `UPDATE memory_entries SET tags=$1, last_update_time=now() WHERE user_id=$2 AND vault_id=$3 AND memory_id=$4 AND entry_id=$5`, nullIfEmpty(tagsJSON), req.UserID, req.VaultID.String(), req.MemoryID, req.EntryID)
	if err != nil {
		return nil, err
	}
	return s.GetMemoryEntryByID(ctx, req.UserID, req.VaultID, req.MemoryID, req.EntryID)
}

func (s *PostgresStorage) DeleteMemoryEntryByID(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, entryID string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `DELETE FROM memory_entries WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3 AND entry_id=$4`, userID, vaultID.String(), memoryID, entryID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if err := writeOutbox(ctx, tx, "delete_entry", entryID, map[string]interface{}{"userId": userID}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- Memory context operations ---

func (s *PostgresStorage) CreateMemoryContext(ctx context.Context, req storage.CreateMemoryContextRequest) (*storage.MemoryContext, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	ctxID := ""
	if req.ContextID != nil && *req.ContextID != "" {
		ctxID = *req.ContextID
	} else {
		ctxID = uuid.New().String()
	}
	var created time.Time
	row := tx.QueryRowContext(ctx, `
        INSERT INTO memory_contexts (user_id, vault_id, memory_id, context_id, context)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING creation_time
    `, req.UserID, req.VaultID.String(), req.MemoryID, ctxID, []byte(req.Context))
	if err := row.Scan(&created); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"userId":       req.UserID,
		"memoryId":     req.MemoryID,
		"contextId":    ctxID,
		"context":      string(req.Context),
		"creationTime": created,
	}
	if err := writeOutbox(ctx, tx, "upsert_context", ctxID, payload); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &storage.MemoryContext{UserID: req.UserID, VaultID: req.VaultID, MemoryID: req.MemoryID, ContextID: ctxID, Context: req.Context, CreationTime: created}, nil
}

func (s *PostgresStorage) GetLatestMemoryContext(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) (*storage.MemoryContext, error) {
	var mc storage.MemoryContext
	mc.UserID = userID
	mc.VaultID = vaultID
	mc.MemoryID = memoryID
	var ctxBytes []byte
	row := s.db.QueryRowContext(ctx, `
        SELECT context_id, context, creation_time
        FROM memory_contexts WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3
        ORDER BY creation_time DESC LIMIT 1
    `, userID, vaultID.String(), memoryID)
	if err := row.Scan(&mc.ContextID, &ctxBytes, &mc.CreationTime); err != nil {
		return nil, err
	}
	mc.Context = json.RawMessage(ctxBytes)
	return &mc, nil
}

func (s *PostgresStorage) DeleteMemoryContextByID(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, contextID string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `DELETE FROM memory_contexts WHERE user_id=$1 AND vault_id=$2 AND memory_id=$3 AND context_id=$4`, userID, vaultID.String(), memoryID, contextID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if err := writeOutbox(ctx, tx, "delete_context", contextID, map[string]interface{}{"userId": userID}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// --- Cross-vault memory association (future) ---

func (s *PostgresStorage) AddMemoryToVault(ctx context.Context, req storage.AddMemoryToVaultRequest) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Validate target vault exists
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM vaults WHERE user_id=$1 AND vault_id=$2`, req.UserID, req.VaultID.String()).Scan(&exists); err != nil {
		return fmt.Errorf("VAULT_NOT_FOUND: %w", err)
	}

	// Locate current vault and title for the memory
	var currentVaultID, title string
	if err := tx.QueryRowContext(ctx, `SELECT vault_id, title FROM memories WHERE user_id=$1 AND memory_id=$2`, req.UserID, req.MemoryID).Scan(&currentVaultID, &title); err != nil {
		return fmt.Errorf("MEMORY_NOT_FOUND: %w", err)
	}

	// No-op if already in target
	if currentVaultID == req.VaultID.String() {
		return tx.Commit()
	}

	// Enforce unique (vault_id, title) within target vault for this user
	var conflict int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM memories WHERE user_id=$1 AND vault_id=$2 AND title=$3`, req.UserID, req.VaultID.String(), title).Scan(&conflict)
	if err == nil {
		return fmt.Errorf("MEMORY_TITLE_CONFLICT: title already exists in target vault")
	}
	if err != sql.ErrNoRows {
		return err
	}

	// Move parent memory
	res, err := tx.ExecContext(ctx, `UPDATE memories SET vault_id=$1 WHERE user_id=$2 AND vault_id=$3 AND memory_id=$4`, req.VaultID.String(), req.UserID, currentVaultID, req.MemoryID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("MEMORY_NOT_FOUND")
	}

	// Move children entries and contexts
	if _, err := tx.ExecContext(ctx, `UPDATE memory_entries SET vault_id=$1 WHERE user_id=$2 AND vault_id=$3 AND memory_id=$4`, req.VaultID.String(), req.UserID, currentVaultID, req.MemoryID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE memory_contexts SET vault_id=$1 WHERE user_id=$2 AND vault_id=$3 AND memory_id=$4`, req.VaultID.String(), req.UserID, currentVaultID, req.MemoryID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStorage) DeleteMemoryFromVault(ctx context.Context, req storage.DeleteMemoryFromVaultRequest) error {
	return s.DeleteMemory(ctx, req.UserID, req.VaultID, req.MemoryID)
}

// --- helpers ---

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
