package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/mycelian/mycelian-memory/server/internal/storage"
)

// SqliteStorage implements storage.Storage using SQLite driver.
// At this stage only HealthCheck is implemented; other methods return ErrNotImplemented
// and will be filled in subsequent tasks.

type SqliteStorage struct {
	db *sql.DB
}

// DB exposes the underlying *sql.DB connection (local-only use case).
func (s *SqliteStorage) DB() *sql.DB {
	return s.db
}

// ErrNotImplemented is returned for methods not yet implemented.
var ErrNotImplemented = errors.New("sqlite adapter: not implemented")

// NewSqliteStorage opens (or creates) a SQLite database file.
func NewSqliteStorage(path string) (storage.Storage, error) {
	db, err := Open(path)
	if err != nil {
		return nil, err
	}
	return NewSqliteStorageWithDB(db)
}

// NewSqliteStorageWithDB allows wiring with an existing connection (used by factory).
func NewSqliteStorageWithDB(db *sql.DB) (storage.Storage, error) {
	return &SqliteStorage{db: db}, nil
}

// --- Health ---

func (s *SqliteStorage) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// --- User operations ---

func (s *SqliteStorage) CreateUser(ctx context.Context, req storage.CreateUserRequest) (*storage.User, error) {
	userID := req.UserID
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `INSERT INTO Users (UserId, Email, DisplayName, TimeZone, Status, CreationTime) VALUES (?,?,?,?,?,?)`,
		userID, req.Email, req.DisplayName, req.TimeZone, "ACTIVE", now)
	if err != nil {
		return nil, err
	}
	return &storage.User{
		UserID:       userID,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		TimeZone:     req.TimeZone,
		Status:       "ACTIVE",
		CreationTime: now,
	}, nil
}
func (s *SqliteStorage) GetUser(ctx context.Context, userID string) (*storage.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT UserId, Email, DisplayName, TimeZone, Status, CreationTime, LastActiveTime FROM Users WHERE UserId = ?`, userID)
	return scanUser(row)
}
func (s *SqliteStorage) GetUserByEmail(ctx context.Context, email string) (*storage.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT UserId, Email, DisplayName, TimeZone, Status, CreationTime, LastActiveTime FROM Users WHERE Email = ?`, email)
	return scanUser(row)
}
func (s *SqliteStorage) UpdateUserLastActive(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE Users SET LastActiveTime = ? WHERE UserId = ?`, time.Now().UTC(), userID)
	return err
}

// --- Memory operations ---

func (s *SqliteStorage) CreateMemory(ctx context.Context, req storage.CreateMemoryRequest) (*storage.Memory, error) {
	memID := uuid.New().String()
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO Memories (UserId, VaultId, MemoryId, MemoryType, Title, Description, CreationTime) VALUES (?,?,?,?,?,?,?)`,
		req.UserID, req.VaultID.String(), memID, req.MemoryType, req.Title, req.Description, now)
	if err != nil {
		return nil, err
	}

	// Insert default context so every memory has at least one snapshot (parity with cloud-dev)
	ctxID := uuid.New().String()
	defaultCtx := `{"activeContext":"This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available."}`
	_, err = s.db.ExecContext(ctx, `INSERT INTO MemoryContexts (UserId, VaultId, MemoryId, Title, ContextId, Context, CreationTime) VALUES (?,?,?,?,?,?,?)`, req.UserID, req.VaultID.String(), memID, req.Title, ctxID, defaultCtx, now)
	if err != nil {
		return nil, err
	}
	publishMemoryCreated(req.UserID, memID)
	return &storage.Memory{
		UserID:       req.UserID,
		VaultID:      req.VaultID,
		MemoryID:     memID,
		MemoryType:   req.MemoryType,
		Title:        req.Title,
		Description:  req.Description,
		CreationTime: now,
	}, nil
}
func (s *SqliteStorage) GetMemory(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) (*storage.Memory, error) {
	row := s.db.QueryRowContext(ctx, `SELECT MemoryType, Title, Description, CreationTime, DeletionScheduledTime FROM Memories WHERE UserId = ? AND VaultId = ? AND MemoryId = ?`, userID, vaultID.String(), memoryID)
	var m storage.Memory
	m.UserID = userID
	m.VaultID = vaultID
	m.MemoryID = memoryID
	err := row.Scan(&m.MemoryType, &m.Title, &m.Description, &m.CreationTime, new(interface{})) // ignore deletion time
	if err != nil {
		return nil, err
	}
	return &m, nil
}
func (s *SqliteStorage) ListMemories(ctx context.Context, userID string, vaultID uuid.UUID) ([]*storage.Memory, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT MemoryId, MemoryType, Title, Description, CreationTime FROM Memories WHERE UserId = ? AND VaultId = ?`, userID, vaultID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
	return out, nil
}
func (s *SqliteStorage) DeleteMemory(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM MemoryEntries WHERE UserId = ? AND VaultId = ? AND MemoryId = ?`, userID, vaultID.String(), memoryID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM Memories WHERE UserId = ? AND VaultId = ? AND MemoryId = ?`, userID, vaultID.String(), memoryID); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// --- Memory entry operations ---

func (s *SqliteStorage) CreateMemoryEntry(ctx context.Context, req storage.CreateMemoryEntryRequest) (*storage.MemoryEntry, error) {
	now := time.Now().UTC()
	entryID := uuid.New().String()

	metaJSON, _ := json.Marshal(req.Metadata)
	tagsJSON, _ := json.Marshal(req.Tags)

	_, err := s.db.ExecContext(ctx, `INSERT INTO MemoryEntries (
		UserId, VaultId, MemoryId, Title, CreationTime, EntryId, RawEntry, Summary, Metadata, Tags)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		req.UserID, req.VaultID.String(), req.MemoryID, "", now, entryID, req.RawEntry, req.Summary, string(metaJSON), string(tagsJSON))
	if err != nil {
		return nil, err
	}
	publishMemoryEntryCreated(req.UserID, req.MemoryID, entryID)
	return &storage.MemoryEntry{
		UserID:       req.UserID,
		MemoryID:     req.MemoryID,
		CreationTime: now,
		EntryID:      entryID,
		RawEntry:     req.RawEntry,
		Summary:      req.Summary,
		Metadata:     req.Metadata,
		Tags:         req.Tags,
	}, nil
}
func (s *SqliteStorage) GetMemoryEntry(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, creationTime time.Time) (*storage.MemoryEntry, error) {
	row := s.db.QueryRowContext(ctx, `SELECT EntryId, RawEntry, Summary, Metadata, Tags FROM MemoryEntries WHERE UserId=? AND VaultId=? AND MemoryId=? AND CreationTime=?`, userID, vaultID.String(), memoryID, creationTime)
	return scanEntry(row, userID, vaultID, memoryID, creationTime)
}
func (s *SqliteStorage) ListMemoryEntries(ctx context.Context, req storage.ListMemoryEntriesRequest) ([]*storage.MemoryEntry, error) {
	q := `SELECT CreationTime, EntryId, RawEntry, Summary, Metadata, Tags FROM MemoryEntries WHERE UserId=? AND VaultId=? AND MemoryId=?`
	args := []interface{}{req.UserID, req.VaultID.String(), req.MemoryID}
	if req.After != nil {
		q += " AND CreationTime > ?"
		args = append(args, *req.After)
	}
	if req.Before != nil {
		q += " AND CreationTime < ?"
		args = append(args, *req.Before)
	}
	q += " ORDER BY CreationTime DESC"
	if req.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", req.Limit)
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*storage.MemoryEntry
	for rows.Next() {
		entry, err := scanEntryRow(rows, req.UserID, req.VaultID, req.MemoryID)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}
func (s *SqliteStorage) CorrectMemoryEntry(ctx context.Context, req storage.CorrectMemoryEntryRequest) (*storage.MemoryEntry, error) {
	return nil, ErrNotImplemented
}
func (s *SqliteStorage) UpdateMemoryEntrySummary(ctx context.Context, req storage.UpdateMemoryEntrySummaryRequest) (*storage.MemoryEntry, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `UPDATE MemoryEntries SET Summary = ?, LastUpdateTime = ? WHERE UserId = ? AND VaultId = ? AND MemoryId = ? AND CreationTime = ?`,
		req.Summary, now, req.UserID, req.VaultID.String(), req.MemoryID, req.CreationTime)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	return s.GetMemoryEntry(ctx, req.UserID, req.VaultID, req.MemoryID, req.CreationTime)
}

func (s *SqliteStorage) UpdateMemoryEntryTags(ctx context.Context, req storage.UpdateMemoryEntryTagsRequest) (*storage.MemoryEntry, error) {
	now := time.Now().UTC()
	tagsJSON, _ := json.Marshal(req.Tags)
	res, err := s.db.ExecContext(ctx, `UPDATE MemoryEntries SET Tags = ?, LastUpdateTime = ? WHERE UserId = ? AND VaultId = ? AND MemoryId = ? AND CreationTime = ?`,
		string(tagsJSON), now, req.UserID, req.VaultID.String(), req.MemoryID, req.CreationTime)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	return s.GetMemoryEntry(ctx, req.UserID, req.VaultID, req.MemoryID, req.CreationTime)
}
func (s *SqliteStorage) SoftDeleteMemoryEntry(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, creationTime time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE MemoryEntries SET DeletionScheduledTime = ? WHERE UserId=? AND VaultId=? AND MemoryId=? AND CreationTime=?`, time.Now().UTC(), userID, vaultID.String(), memoryID, creationTime)
	return err
}

// --- Memory context ---

func (s *SqliteStorage) CreateMemoryContext(ctx context.Context, req storage.CreateMemoryContextRequest) (*storage.MemoryContext, error) {
	now := time.Now().UTC()
	ctxID := req.ContextID
	if ctxID == nil {
		id := uuid.New().String()
		ctxID = &id
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO MemoryContexts (UserId, VaultId, MemoryId, Title, ContextId, Context, CreationTime) VALUES (?,?,?,?,?,?,?)`,
		req.UserID, req.VaultID.String(), req.MemoryID, "", *ctxID, string(req.Context), now)
	if err != nil {
		return nil, err
	}
	return &storage.MemoryContext{
		UserID:       req.UserID,
		VaultID:      req.VaultID,
		MemoryID:     req.MemoryID,
		ContextID:    *ctxID,
		Context:      req.Context,
		CreationTime: now,
	}, nil
}
func (s *SqliteStorage) GetLatestMemoryContext(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) (*storage.MemoryContext, error) {
	row := s.db.QueryRowContext(ctx, `SELECT ContextId, Context, CreationTime FROM MemoryContexts WHERE UserId=? AND VaultId=? AND MemoryId=? ORDER BY CreationTime DESC LIMIT 1`, userID, vaultID.String(), memoryID)
	var ctxID string
	var ctxJSON string
	var creation time.Time
	if err := row.Scan(&ctxID, &ctxJSON, &creation); err != nil {
		return nil, err
	}
	return &storage.MemoryContext{
		UserID:       userID,
		VaultID:      vaultID,
		MemoryID:     memoryID,
		ContextID:    ctxID,
		Context:      json.RawMessage(ctxJSON),
		CreationTime: creation,
	}, nil
}

// --- Vault operations ---

func (s *SqliteStorage) CreateVault(ctx context.Context, req storage.CreateVaultRequest) (*storage.Vault, error) {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO Vaults (UserId, VaultId, Title, Description, CreationTime) VALUES (?,?,?,?,?)`,
		req.UserID, req.VaultID.String(), req.Title, req.Description, now)
	if err != nil {
		return nil, err
	}
	return &storage.Vault{
		UserID:       req.UserID,
		VaultID:      req.VaultID,
		Title:        req.Title,
		Description:  req.Description,
		CreationTime: now,
	}, nil
}

func (s *SqliteStorage) GetVault(ctx context.Context, userID string, vaultID uuid.UUID) (*storage.Vault, error) {
	row := s.db.QueryRowContext(ctx, `SELECT Title, Description, CreationTime FROM Vaults WHERE UserId=? AND VaultId=?`, userID, vaultID.String())
	var v storage.Vault
	v.UserID = userID
	v.VaultID = vaultID
	if err := row.Scan(&v.Title, &v.Description, &v.CreationTime); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *SqliteStorage) ListVaults(ctx context.Context, userID string) ([]*storage.Vault, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT VaultId, Title, Description, CreationTime FROM Vaults WHERE UserId=? ORDER BY CreationTime DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*storage.Vault
	for rows.Next() {
		var v storage.Vault
		v.UserID = userID
		if err := rows.Scan(&v.VaultID, &v.Title, &v.Description, &v.CreationTime); err != nil {
			return nil, err
		}
		list = append(list, &v)
	}
	return list, nil
}

func (s *SqliteStorage) GetVaultByTitle(ctx context.Context, userID string, title string) (*storage.Vault, error) {
	row := s.db.QueryRowContext(ctx, `SELECT VaultId, Description, CreationTime FROM Vaults WHERE UserId=? AND Title=?`, userID, title)
	var v storage.Vault
	v.UserID = userID
	v.Title = title
	if err := row.Scan(&v.VaultID, &v.Description, &v.CreationTime); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *SqliteStorage) GetMemoryByTitle(ctx context.Context, userID string, vaultID uuid.UUID, title string) (*storage.Memory, error) {
	row := s.db.QueryRowContext(ctx, `SELECT MemoryId, MemoryType, Description, CreationTime FROM Memories WHERE UserId=? AND VaultId=? AND Title=?`, userID, vaultID.String(), title)
	var m storage.Memory
	m.UserID = userID
	m.VaultID = vaultID
	m.Title = title
	if err := row.Scan(&m.MemoryID, &m.MemoryType, &m.Description, &m.CreationTime); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *SqliteStorage) DeleteVault(ctx context.Context, userID string, vaultID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM Vaults WHERE UserId=? AND VaultId=?`, userID, vaultID.String())
	return err
}

func (s *SqliteStorage) AddMemoryToVault(ctx context.Context, req storage.AddMemoryToVaultRequest) error {
	return ErrNotImplemented
}

func (s *SqliteStorage) DeleteMemoryFromVault(ctx context.Context, req storage.DeleteMemoryFromVaultRequest) error {
	return ErrNotImplemented
}

// helper
func scanUser(row *sql.Row) (*storage.User, error) {
	var u storage.User
	err := row.Scan(&u.UserID, &u.Email, &u.DisplayName, &u.TimeZone, &u.Status, &u.CreationTime, &u.LastActiveTime)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func scanEntry(row *sql.Row, userID string, vaultID uuid.UUID, memoryID string, creation time.Time) (*storage.MemoryEntry, error) {
	var entry storage.MemoryEntry
	entry.UserID = userID
	entry.VaultID = vaultID
	entry.MemoryID = memoryID
	entry.CreationTime = creation
	var metaStr, tagsStr sql.NullString
	if err := row.Scan(&entry.EntryID, &entry.RawEntry, &entry.Summary, &metaStr, &tagsStr); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(metaStr.String), &entry.Metadata)
	json.Unmarshal([]byte(tagsStr.String), &entry.Tags)
	return &entry, nil
}

func scanEntryRow(rows *sql.Rows, userID string, vaultID uuid.UUID, memoryID string) (*storage.MemoryEntry, error) {
	var entry storage.MemoryEntry
	entry.UserID = userID
	entry.VaultID = vaultID
	entry.MemoryID = memoryID
	var metaStr, tagsStr sql.NullString
	if err := rows.Scan(&entry.CreationTime, &entry.EntryID, &entry.RawEntry, &entry.Summary, &metaStr, &tagsStr); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(metaStr.String), &entry.Metadata)
	json.Unmarshal([]byte(tagsStr.String), &entry.Tags)
	return &entry, nil
}
