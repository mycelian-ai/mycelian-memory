package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"

	"github.com/rs/zerolog/log"
)

// SpannerStorage implements the Storage interface using Google Cloud Spanner
type SpannerStorage struct {
	client *spanner.Client
}

// NewSpannerStorage creates a new SpannerStorage instance
func NewSpannerStorage(client *spanner.Client) Storage {
	return &SpannerStorage{
		client: client,
	}
}

// CreateUser creates a new user with client-generated UUID
func (s *SpannerStorage) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	userID := req.UserID

	mutation := spanner.Insert("Users",
		[]string{"UserId", "Email", "DisplayName", "TimeZone", "Status", "CreationTime"},
		[]interface{}{userID, req.Email, req.DisplayName, req.TimeZone, "ACTIVE", spanner.CommitTimestamp},
	)

	_, err := s.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Return user with known UUID - no additional query needed
	return &User{
		UserID:       userID,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		TimeZone:     req.TimeZone,
		Status:       "ACTIVE",
		CreationTime: time.Now(), // Approximate timestamp
	}, nil
}

// GetUser retrieves a user by ID
func (s *SpannerStorage) GetUser(ctx context.Context, userID string) (*User, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, Email, DisplayName, TimeZone, Status, CreationTime, LastActiveTime 
			  FROM Users WHERE UserId = @userId`,
		Params: map[string]interface{}{
			"userId": userID,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user := &User{}
	err = row.Columns(&user.UserID, &user.Email, &user.DisplayName, &user.TimeZone,
		&user.Status, &user.CreationTime, &user.LastActiveTime)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email address
func (s *SpannerStorage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, Email, DisplayName, TimeZone, Status, CreationTime, LastActiveTime 
			  FROM Users WHERE Email = @email AND Status = 'ACTIVE'`,
		Params: map[string]interface{}{
			"email": email,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("user not found with email: %s", email)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	user := &User{}
	err = row.Columns(&user.UserID, &user.Email, &user.DisplayName, &user.TimeZone,
		&user.Status, &user.CreationTime, &user.LastActiveTime)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return user, nil
}

// UpdateUserLastActive updates the user's last active timestamp
func (s *SpannerStorage) UpdateUserLastActive(ctx context.Context, userID string) error {
	mutation := spanner.Update("Users",
		[]string{"UserId", "LastActiveTime"},
		[]interface{}{userID, spanner.CommitTimestamp},
	)

	_, err := s.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return fmt.Errorf("failed to update user last active: %w", err)
	}

	return nil
}

// CreateMemory creates a new memory with client-generated UUID
func (s *SpannerStorage) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error) {
	// Generate UUIDs locally – goroutine-safe, function-scoped
	memoryID := uuid.New().String()
	contextID := uuid.New().String()

	// Prepare mutations upfront so they share the same commit timestamp.
	memMut := spanner.Insert("Memories",
		[]string{"MemoryId", "UserId", "VaultId", "MemoryType", "Title", "Description", "CreationTime"},
		[]interface{}{memoryID, req.UserID, req.VaultID.String(), req.MemoryType, req.Title, req.Description, spanner.CommitTimestamp},
	)

	defaultJSON := `{"activeContext":"This is default context that's created with the memory. Instructions for AI Agent: Provide relevant context as soon as it's available."}`
	defaultCtx := spanner.NullJSON{Value: defaultJSON, Valid: true}
	ctxMut := spanner.Insert("MemoryContexts",
		[]string{"UserId", "VaultId", "MemoryId", "ContextId", "Context", "CreationTime"},
		[]interface{}{req.UserID, req.VaultID.String(), memoryID, contextID, defaultCtx, spanner.CommitTimestamp},
	)

	// Execute both inserts atomically so that a newly-created memory always has a default context.
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{memMut, ctxMut})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create memory (with default context): %w", err)
	}

	// Return memory with known UUID – eliminates race conditions for caller logic.
	return &Memory{
		MemoryID:     memoryID,
		UserID:       req.UserID,
		VaultID:      req.VaultID,
		MemoryType:   req.MemoryType,
		Title:        req.Title,
		Description:  req.Description,
		CreationTime: time.Now(), // Approximate timestamp
	}, nil
}

// GetMemory retrieves a memory by user ID, vault ID and memory ID
func (s *SpannerStorage) GetMemory(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) (*Memory, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, MemoryId, MemoryType, Title, Description, CreationTime 
			  FROM Memories WHERE UserId = @userId AND VaultId = @vaultId AND MemoryId = @memoryId`,
		Params: map[string]interface{}{
			"userId":   userID,
			"vaultId":  vaultID.String(),
			"memoryId": memoryID,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("memory not found: %s/%s", userID, memoryID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	memory := &Memory{}
	err = row.Columns(&memory.UserID, &memory.MemoryID, &memory.MemoryType,
		&memory.Title, &memory.Description, &memory.CreationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to scan memory: %w", err)
	}
	memory.VaultID = vaultID

	return memory, nil
}

// ListMemories retrieves all memories for a user within a vault
func (s *SpannerStorage) ListMemories(ctx context.Context, userID string, vaultID uuid.UUID) ([]*Memory, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, MemoryId, MemoryType, Title, Description, CreationTime 
			  FROM Memories WHERE UserId = @userId AND VaultId = @vaultId ORDER BY CreationTime DESC`,
		Params: map[string]interface{}{
			"userId":  userID,
			"vaultId": vaultID.String(),
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var memories []*Memory
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate memories: %w", err)
		}

		memory := &Memory{}
		err = row.Columns(&memory.UserID, &memory.MemoryID, &memory.MemoryType,
			&memory.Title, &memory.Description, &memory.CreationTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		memory.VaultID = vaultID

		memories = append(memories, memory)
	}

	return memories, nil
}

// DeleteMemory deletes a memory within a vault
func (s *SpannerStorage) DeleteMemory(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) error {
	mutation := spanner.Delete("Memories",
		spanner.Key{userID, vaultID.String(), memoryID},
	)

	_, err := s.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	return nil
}

// --- Vault operations ---

// CreateVault inserts a new vault for a user
func (s *SpannerStorage) CreateVault(ctx context.Context, req CreateVaultRequest) (*Vault, error) {
	now := time.Now()
	mut := spanner.Insert("Vaults",
		[]string{"UserId", "VaultId", "Title", "Description", "CreationTime"},
		[]interface{}{req.UserID, req.VaultID.String(), req.Title, req.Description, spanner.CommitTimestamp},
	)

	if _, err := s.client.Apply(ctx, []*spanner.Mutation{mut}); err != nil {
		return nil, err
	}
	return &Vault{
		UserID:       req.UserID,
		VaultID:      req.VaultID,
		Title:        req.Title,
		Description:  req.Description,
		CreationTime: now,
	}, nil
}

// GetVault fetches a single vault by composite key
func (s *SpannerStorage) GetVault(ctx context.Context, userID string, vaultID uuid.UUID) (*Vault, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT Title, Description, CreationTime FROM Vaults WHERE UserId=@u AND VaultId=@v`,
		Params: map[string]interface{}{"u": userID, "v": vaultID.String()},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("vault not found")
	}
	if err != nil {
		return nil, err
	}
	var v Vault
	v.UserID = userID
	v.VaultID = vaultID
	if err := row.Columns(&v.Title, &v.Description, &v.CreationTime); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVaults returns all vaults for a user ordered by creation time desc
func (s *SpannerStorage) ListVaults(ctx context.Context, userID string) ([]*Vault, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT VaultId, Title, Description, CreationTime FROM Vaults WHERE UserId=@u ORDER BY CreationTime DESC`,
		Params: map[string]interface{}{"u": userID},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []*Vault
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var v Vault
		v.UserID = userID
		var vaultIDStr string
		if err := row.Columns(&vaultIDStr, &v.Title, &v.Description, &v.CreationTime); err != nil {
			return nil, err
		}
		// Parse UUID string stored in Spanner
		id, err := uuid.Parse(vaultIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid vaultId stored: %w", err)
		}
		v.VaultID = id
		out = append(out, &v)
	}
	return out, nil
}

// GetVaultByTitle fetches a vault by userID and unique title.
func (s *SpannerStorage) GetVaultByTitle(ctx context.Context, userID string, title string) (*Vault, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT VaultId, Description, CreationTime FROM Vaults WHERE UserId = @uid AND Title = @title LIMIT 1`,
		Params: map[string]interface{}{"uid": userID, "title": title},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("vault not found by title: %s", title)
	}
	if err != nil {
		return nil, fmt.Errorf("query vault by title: %w", err)
	}
	var v Vault
	v.UserID = userID
	var vaultIDStr string
	if err := row.Columns(&vaultIDStr, &v.Description, &v.CreationTime); err != nil {
		return nil, err
	}
	// Parse UUID string to uuid.UUID type
	id, err := uuid.Parse(vaultIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid vaultId stored: %w", err)
	}
	v.VaultID = id
	v.Title = title
	return &v, nil
}

// GetMemoryByTitle fetches a memory by vaultID + title.
func (s *SpannerStorage) GetMemoryByTitle(ctx context.Context, userID string, vaultID uuid.UUID, title string) (*Memory, error) {
	stmt := spanner.Statement{
		SQL:    `SELECT MemoryId, MemoryType, Description, CreationTime FROM Memories WHERE UserId = @uid AND VaultId = @vid AND Title = @title LIMIT 1`,
		Params: map[string]interface{}{"uid": userID, "vid": vaultID.String(), "title": title},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("memory not found by title: %s", title)
	}
	if err != nil {
		return nil, fmt.Errorf("query memory by title: %w", err)
	}
	var m Memory
	m.UserID = userID
	m.VaultID = vaultID
	m.Title = title
	if err := row.Columns(&m.MemoryID, &m.MemoryType, &m.Description, &m.CreationTime); err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteVault removes a vault and cascades via FK
func (s *SpannerStorage) DeleteVault(ctx context.Context, userID string, vaultID uuid.UUID) error {
	mut := spanner.Delete("Vaults", spanner.Key{userID, vaultID.String()})
	_, err := s.client.Apply(ctx, []*spanner.Mutation{mut})
	return err
}

// AddMemoryToVault associates an existing memory with a vault (not implemented in Spanner yet)
func (s *SpannerStorage) AddMemoryToVault(ctx context.Context, req AddMemoryToVaultRequest) error {
	return fmt.Errorf("AddMemoryToVault not implemented")
}

// DeleteMemoryFromVault disassociates a memory from a vault (not implemented in Spanner yet)
func (s *SpannerStorage) DeleteMemoryFromVault(ctx context.Context, req DeleteMemoryFromVaultRequest) error {
	return fmt.Errorf("DeleteMemoryFromVault not implemented")
}

// CreateMemoryEntry creates a new memory entry with client-generated UUID
func (s *SpannerStorage) CreateMemoryEntry(ctx context.Context, req CreateMemoryEntryRequest) (*MemoryEntry, error) {
	// Generate UUID locally - each function call gets isolated UUID
	entryID := uuid.New().String()

	// Convert metadata to JSON if provided
	var metadataJSON spanner.NullJSON
	if req.Metadata != nil {
		metadataJSON = spanner.NullJSON{Value: req.Metadata, Valid: true}
	}

	// Convert tags to JSON if provided
	var tagsJSON spanner.NullJSON
	if req.Tags != nil {
		tagsJSON = spanner.NullJSON{Value: req.Tags, Valid: true}
	}

	mutation := spanner.Insert("MemoryEntries",
		[]string{"EntryId", "UserId", "VaultId", "MemoryId", "CreationTime", "RawEntry",
			"Summary", "Metadata", "Tags", "ExpirationTime"},
		[]interface{}{entryID, req.UserID, req.VaultID.String(), req.MemoryID, spanner.CommitTimestamp,
			req.RawEntry, req.Summary,
			metadataJSON, tagsJSON, req.ExpirationTime},
	)

	_, err := s.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return nil, fmt.Errorf("failed to create memory entry: %w", err)
	}

	// Get just the CreationTime using the known EntryId (minimal query)
	stmt := spanner.Statement{
		SQL: `SELECT CreationTime FROM MemoryEntries WHERE EntryId = @entryId`,
		Params: map[string]interface{}{
			"entryId": entryID,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("failed to retrieve created memory entry timestamp")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query created memory entry timestamp: %w", err)
	}

	var actualCreationTime time.Time
	err = row.Columns(&actualCreationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to scan creation time: %w", err)
	}

	// Return entry with known UUID and exact timestamp
	return &MemoryEntry{
		EntryID:        entryID,
		UserID:         req.UserID,
		VaultID:        req.VaultID,
		MemoryID:       req.MemoryID,
		CreationTime:   actualCreationTime, // Exact Spanner timestamp
		RawEntry:       req.RawEntry,
		Summary:        req.Summary,
		Metadata:       req.Metadata,
		Tags:           req.Tags,
		ExpirationTime: req.ExpirationTime,
	}, nil
}

// GetMemoryEntry retrieves a single entry within a vault
func (s *SpannerStorage) GetMemoryEntry(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, creationTime time.Time) (*MemoryEntry, error) {
	stmt := spanner.Statement{
		SQL: `SELECT EntryId, RawEntry, Summary, Metadata, Tags, ExpirationTime, CorrectionTime, CorrectedEntryMemoryId, CorrectedEntryCreationTime, CorrectionReason, LastUpdateTime
			  FROM MemoryEntries WHERE UserId=@userId AND VaultId=@vaultId AND MemoryId=@memoryId AND CreationTime=@creation`,
		Params: map[string]interface{}{
			"userId":   userID,
			"vaultId":  vaultID.String(),
			"memoryId": memoryID,
			"creation": creationTime,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("memory entry not found: %s/%s/%v", userID, memoryID, creationTime)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get memory entry: %w", err)
	}

	entry := &MemoryEntry{}
	var metadataJSON, tagsJSON spanner.NullJSON
	err = row.Columns(&entry.EntryID, &entry.RawEntry, &entry.Summary,
		&metadataJSON, &tagsJSON, &entry.ExpirationTime, &entry.CorrectionTime, &entry.CorrectedEntryMemoryId,
		&entry.CorrectedEntryCreationTime, &entry.CorrectionReason, &entry.LastUpdateTime)
	if err != nil {
		return nil, fmt.Errorf("failed to scan memory entry: %w", err)
	}

	// Convert JSON back to map
	if metadataJSON.Valid {
		if m, err := convertJSONToMap(metadataJSON.Value); err == nil {
			entry.Metadata = m
		} else {
			log.Warn().Err(err).Msg("failed to parse metadata JSON")
		}
	}
	if tagsJSON.Valid {
		if t, err := convertJSONToMap(tagsJSON.Value); err == nil {
			entry.Tags = t
		} else {
			log.Warn().Err(err).Msg("failed to parse tags JSON")
		}
	}

	entry.VaultID = vaultID
	return entry, nil
}

// GetMemoryEntryByID retrieves a single entry by entryId
func (s *SpannerStorage) GetMemoryEntryByID(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, entryID string) (*MemoryEntry, error) {
	// Query by EntryId (globally unique) and return full row
	stmt := spanner.Statement{
		SQL: `SELECT UserId, VaultId, MemoryId, CreationTime, EntryId, RawEntry, Summary, Metadata, Tags,
                     ExpirationTime, CorrectionTime, CorrectedEntryMemoryId, CorrectedEntryCreationTime,
                     CorrectionReason, LastUpdateTime
              FROM MemoryEntries WHERE EntryId=@entryId LIMIT 1`,
		Params: map[string]interface{}{"entryId": entryID},
	}
	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("memory entry not found: %s", entryID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get memory entry by id: %w", err)
	}
	var uid, vid, mid string
	entry := &MemoryEntry{}
	var metadataJSON, tagsJSON spanner.NullJSON
	if err := row.Columns(&uid, &vid, &mid, &entry.CreationTime, &entry.EntryID, &entry.RawEntry, &entry.Summary,
		&metadataJSON, &tagsJSON, &entry.ExpirationTime, &entry.CorrectionTime, &entry.CorrectedEntryMemoryId,
		&entry.CorrectedEntryCreationTime, &entry.CorrectionReason, &entry.LastUpdateTime); err != nil {
		return nil, fmt.Errorf("failed to scan memory entry: %w", err)
	}
	if v, err := uuid.Parse(vid); err == nil {
		entry.VaultID = v
	}
	entry.UserID = uid
	entry.MemoryID = mid
	if metadataJSON.Valid {
		if m, err := convertJSONToMap(metadataJSON.Value); err == nil {
			entry.Metadata = m
		} else {
			log.Warn().Err(err).Msg("failed to parse metadata JSON")
		}
	}
	if tagsJSON.Valid {
		if t, err := convertJSONToMap(tagsJSON.Value); err == nil {
			entry.Tags = t
		} else {
			log.Warn().Err(err).Msg("failed to parse tags JSON")
		}
	}
	return entry, nil
}

// ListMemoryEntries retrieves memory entries with optional filtering
func (s *SpannerStorage) ListMemoryEntries(ctx context.Context, req ListMemoryEntriesRequest) ([]*MemoryEntry, error) {
	query := `SELECT UserId, VaultId, MemoryId, CreationTime, EntryId, RawEntry, Summary, 
                 Metadata, Tags, CorrectionTime, CorrectedEntryMemoryId, 
                 CorrectedEntryCreationTime, CorrectionReason, LastUpdateTime,
                 ExpirationTime 
              FROM MemoryEntries 
              WHERE UserId = @userId AND VaultId = @vaultId AND MemoryId = @memoryId`
	params := map[string]interface{}{
		"userId":   req.UserID,
		"vaultId":  req.VaultID.String(),
		"memoryId": req.MemoryID,
	}

	// Add time filtering if specified
	if req.Before != nil {
		query += " AND CreationTime < @before"
		params["before"] = *req.Before
	}
	if req.After != nil {
		query += " AND CreationTime > @after"
		params["after"] = *req.After
	}

	// Order by creation time (newest first)
	query += " ORDER BY CreationTime DESC"

	// Add limit if specified
	if req.Limit > 0 {
		query += " LIMIT @limit"
		params["limit"] = req.Limit
	}

	stmt := spanner.Statement{
		SQL:    query,
		Params: params,
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var entries []*MemoryEntry
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate memory entries: %w", err)
		}

		entry := &MemoryEntry{}
		var vaultIDStr string
		var metadataJSON, tagsJSON spanner.NullJSON
		err = row.Columns(&entry.UserID, &vaultIDStr, &entry.MemoryID, &entry.CreationTime,
			&entry.EntryID, &entry.RawEntry, &entry.Summary,
			&metadataJSON, &tagsJSON, &entry.CorrectionTime, &entry.CorrectedEntryMemoryId,
			&entry.CorrectedEntryCreationTime, &entry.CorrectionReason,
			&entry.LastUpdateTime, &entry.ExpirationTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory entry: %w", err)
		}

		// convert vaultID string
		if v, err := uuid.Parse(vaultIDStr); err == nil {
			entry.VaultID = v
		}

		// Convert JSON back to map
		if metadataJSON.Valid {
			if m, err := convertJSONToMap(metadataJSON.Value); err == nil {
				entry.Metadata = m
			} else {
				log.Warn().Err(err).Msg("failed to parse metadata JSON")
			}
		}
		if tagsJSON.Valid {
			if t, err := convertJSONToMap(tagsJSON.Value); err == nil {
				entry.Tags = t
			} else {
				log.Warn().Err(err).Msg("failed to parse tags JSON")
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

//
// 🔒 CRITICAL SYSTEM FILE - Memory State Validation
// ⚠️  Changes to validation logic affect data integrity
// 🛡️  All mutations must go through validateEntryMutable()
//

// CorrectMemoryEntry creates a correction entry with transactional integrity
func (s *SpannerStorage) CorrectMemoryEntry(ctx context.Context, req CorrectMemoryEntryRequest) (*MemoryEntry, error) {
	var correctionEntry *MemoryEntry

	// Use ReadWriteTransaction for atomic correction operations
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {

		// Step 1: Verify original entry exists and is correctable
		checkStmt := spanner.Statement{
			SQL: `SELECT UserId, MemoryId, CreationTime, EntryId, RawEntry, CorrectionTime
                  FROM MemoryEntries 
                  WHERE UserId = @userId AND MemoryId = @memoryId AND CreationTime = @creationTime`,
			Params: map[string]interface{}{
				"userId":       req.UserID,
				"memoryId":     req.MemoryID,
				"creationTime": req.OriginalCreationTime,
			},
		}

		iter := txn.Query(ctx, checkStmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			return fmt.Errorf("ENTRY_NOT_FOUND: entry does not exist")
		}
		if err != nil {
			return fmt.Errorf("failed to check original entry: %w", err)
		}

		var originalEntry struct {
			UserID         string
			MemoryID       string
			CreationTime   time.Time
			EntryID        string
			RawEntry       string
			CorrectionTime *time.Time
		}

		err = row.Columns(&originalEntry.UserID, &originalEntry.MemoryID,
			&originalEntry.CreationTime, &originalEntry.EntryID, &originalEntry.RawEntry,
			&originalEntry.CorrectionTime)
		if err != nil {
			return fmt.Errorf("failed to scan original entry: %w", err)
		}

		// 🔒 INVARIANT: Entry must be mutable (not corrected, not deleted)
		if originalEntry.CorrectionTime != nil {
			return fmt.Errorf("IMMUTABILITY_VIOLATION: entry was already corrected at %v",
				*originalEntry.CorrectionTime)
		}

		// hard-deleted rows no longer exist; no soft-delete guard needed

		// Step 2: Create the correction entry (new entry with corrected content)
		now := time.Now()
		correctionEntry = &MemoryEntry{
			UserID:       req.UserID,
			MemoryID:     req.MemoryID,
			CreationTime: now,
			EntryID:      req.CorrectedEntryID,
			RawEntry:     req.CorrectedContent,
			Summary:      req.CorrectedSummary,
			Metadata:     req.Metadata,
			Tags:         req.Tags,
		}

		// Convert metadata to JSON if provided
		var metadataJSON spanner.NullJSON
		if req.Metadata != nil {
			metadataJSON = spanner.NullJSON{Value: req.Metadata, Valid: true}
		}

		// Convert tags to JSON if provided
		var tagsJSON spanner.NullJSON
		if req.Tags != nil {
			tagsJSON = spanner.NullJSON{Value: req.Tags, Valid: true}
		}

		// Insert the correction entry
		correctionMutation := spanner.Insert("MemoryEntries",
			[]string{"UserId", "MemoryId", "CreationTime", "EntryId", "RawEntry",
				"Summary", "Metadata", "Tags"},
			[]interface{}{req.UserID, req.MemoryID,
				spanner.CommitTimestamp, correctionEntry.EntryID, correctionEntry.RawEntry,
				correctionEntry.Summary, metadataJSON, tagsJSON},
		)

		// Step 3: Update original entry to mark it as corrected
		originalUpdateMutation := spanner.Update("MemoryEntries",
			[]string{"UserId", "MemoryId", "CreationTime", "CorrectionTime",
				"CorrectedEntryMemoryId", "CorrectedEntryCreationTime", "CorrectionReason"},
			[]interface{}{originalEntry.UserID, originalEntry.MemoryID,
				originalEntry.CreationTime, spanner.CommitTimestamp,
				req.MemoryID, spanner.CommitTimestamp, req.CorrectionReason},
		)

		// Apply both mutations atomically
		err = txn.BufferWrite([]*spanner.Mutation{correctionMutation, originalUpdateMutation})
		if err != nil {
			return fmt.Errorf("failed to apply correction mutations: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Set the commit timestamp
	correctionEntry.CreationTime = time.Now()
	return correctionEntry, nil
}

// UpdateMemoryEntrySummary updates only the summary field (only allowed update)
func (s *SpannerStorage) UpdateMemoryEntrySummary(ctx context.Context, req UpdateMemoryEntrySummaryRequest) (*MemoryEntry, error) {

	// Use ReadWriteTransaction for validation + update
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {

		// Verify entry exists and is updatable (lookup by EntryId)
		checkStmt := spanner.Statement{
			SQL: `SELECT CorrectionTime FROM MemoryEntries WHERE EntryId = @entryId`,
			Params: map[string]interface{}{
				"entryId": req.EntryID,
			},
		}

		iter := txn.Query(ctx, checkStmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			return fmt.Errorf("ENTRY_NOT_FOUND: entry does not exist")
		}
		if err != nil {
			return fmt.Errorf("failed to check entry: %w", err)
		}

		var correctionTime *time.Time
		err = row.Columns(&correctionTime)
		if err != nil {
			return fmt.Errorf("failed to scan entry state: %w", err)
		}

		// 🔒 INVARIANT: Cannot update corrected or deleted entries
		if correctionTime != nil {
			return fmt.Errorf("IMMUTABILITY_VIOLATION: cannot update summary of corrected entry")
		}

		// hard-deleted rows do not appear here

		// Update summary and lastUpdateTime using DML keyed by EntryId
		stmt := spanner.Statement{
			SQL: `UPDATE MemoryEntries
                  SET Summary = @summary,
                      LastUpdateTime = PENDING_COMMIT_TIMESTAMP()
                  WHERE EntryId = @entryId`,
			Params: map[string]interface{}{
				"summary": req.Summary,
				"entryId": req.EntryID,
			},
		}
		_, err = txn.Update(ctx, stmt)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Return updated entry
	return s.GetMemoryEntryByID(ctx, req.UserID, req.VaultID, req.MemoryID, req.EntryID)
}

// UpdateMemoryEntryTags updates only the tags field (mutable operational metadata)
func (s *SpannerStorage) UpdateMemoryEntryTags(ctx context.Context, req UpdateMemoryEntryTagsRequest) (*MemoryEntry, error) {

	// Use ReadWriteTransaction for validation + update
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {

		// Verify entry exists and is updatable (lookup by EntryId)
		checkStmt := spanner.Statement{
			SQL: `SELECT CorrectionTime FROM MemoryEntries WHERE EntryId = @entryId`,
			Params: map[string]interface{}{
				"entryId": req.EntryID,
			},
		}

		iter := txn.Query(ctx, checkStmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err == iterator.Done {
			return fmt.Errorf("ENTRY_NOT_FOUND: entry does not exist")
		}
		if err != nil {
			return fmt.Errorf("failed to check entry: %w", err)
		}

		var correctionTime *time.Time
		err = row.Columns(&correctionTime)
		if err != nil {
			return fmt.Errorf("failed to scan entry state: %w", err)
		}

		// 🔒 INVARIANT: Cannot update corrected or deleted entries
		if correctionTime != nil {
			return fmt.Errorf("IMMUTABILITY_VIOLATION: cannot update tags of corrected entry")
		}

		// hard-deleted rows do not appear here

		// Convert tags to JSON
		var tagsJSON spanner.NullJSON
		if req.Tags != nil {
			tagsJSON = spanner.NullJSON{Value: req.Tags, Valid: true}
		}

		// Update tags and lastUpdateTime using DML keyed by EntryId
		stmt := spanner.Statement{
			SQL: `UPDATE MemoryEntries
                  SET Tags = @tags,
                      LastUpdateTime = PENDING_COMMIT_TIMESTAMP()
                  WHERE EntryId = @entryId`,
			Params: map[string]interface{}{
				"tags":    tagsJSON,
				"entryId": req.EntryID,
			},
		}
		_, err = txn.Update(ctx, stmt)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Return updated entry
	return s.GetMemoryEntryByID(ctx, req.UserID, req.VaultID, req.MemoryID, req.EntryID)
}

// SoftDeleteMemoryEntry marks an entry for eventual deletion
// DeleteMemoryEntryByID performs a hard delete by external entryId
func (s *SpannerStorage) DeleteMemoryEntryByID(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, entryID string) error {
	stmt := spanner.Statement{
		SQL: `DELETE FROM MemoryEntries WHERE UserId=@userId AND VaultId=@vaultId AND MemoryId=@memoryId AND EntryId=@entryId`,
		Params: map[string]interface{}{
			"userId":   userID,
			"vaultId":  vaultID.String(),
			"memoryId": memoryID,
			"entryId":  entryID,
		},
	}
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}

// HealthCheck performs a health check by executing a simple query
func (s *SpannerStorage) HealthCheck(ctx context.Context) error {
	// Query Users table to ensure schema is set up correctly
	stmt := spanner.Statement{
		SQL: "SELECT COUNT(*) FROM Users",
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err := iter.Next()
	if err != nil && err != iterator.Done {
		return fmt.Errorf("spanner health check failed: %w", err)
	}

	return nil
}

func (s *SpannerStorage) CreateMemoryContext(ctx context.Context, req CreateMemoryContextRequest) (*MemoryContext, error) {
	// Validate
	if req.UserID == "" || req.MemoryID == "" {
		return nil, fmt.Errorf("userID and memoryID are required")
	}
	if len(req.Context) == 0 {
		return nil, fmt.Errorf("context payload is required")
	}

	contextID := ""
	if req.ContextID != nil && *req.ContextID != "" {
		contextID = *req.ContextID
	} else {
		contextID = uuid.New().String()
	}

	ctxJSON := spanner.NullJSON{Value: string(req.Context), Valid: true}

	mutation := spanner.Insert("MemoryContexts",
		[]string{"UserId", "VaultId", "MemoryId", "ContextId", "Context", "CreationTime"},
		[]interface{}{req.UserID, req.VaultID.String(), req.MemoryID, contextID, ctxJSON, spanner.CommitTimestamp},
	)

	_, err := s.client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return nil, fmt.Errorf("failed to insert memory context: %w", err)
	}

	return &MemoryContext{
		UserID:       req.UserID,
		VaultID:      req.VaultID,
		MemoryID:     req.MemoryID,
		ContextID:    contextID,
		Context:      req.Context,
		CreationTime: time.Now(), // approx
	}, nil
}

// GetLatestMemoryContext returns the most recent context snapshot for a memory
func (s *SpannerStorage) GetLatestMemoryContext(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) (*MemoryContext, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserId, VaultId, MemoryId, ContextId, Context, CreationTime
			   FROM MemoryContexts
			   WHERE UserId=@userId AND VaultId=@vaultId AND MemoryId=@memoryId
			   ORDER BY CreationTime DESC
			   LIMIT 1`,
		Params: map[string]interface{}{
			"userId":   userID,
			"vaultId":  vaultID.String(),
			"memoryId": memoryID,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("context not found for memory: %s/%s", userID, memoryID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest memory context: %w", err)
	}

	var vaultIDStr string
	var ctxField spanner.NullJSON
	mc := &MemoryContext{}
	err = row.Columns(&mc.UserID, &vaultIDStr, &mc.MemoryID, &mc.ContextID, &ctxField, &mc.CreationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to scan memory context: %w", err)
	}
	if v, errParse := uuid.Parse(vaultIDStr); errParse == nil {
		mc.VaultID = v
	}

	if ctxField.Valid {
		switch v := ctxField.Value.(type) {
		case string:
			mc.Context = []byte(v)
		case []byte:
			mc.Context = v
		default:
			// fallback: marshal
			if b, err := json.Marshal(v); err == nil {
				mc.Context = b
			}
		}
	}

	return mc, nil
}

// DeleteMemoryContextByID performs a hard delete of a context snapshot by its contextId
func (s *SpannerStorage) DeleteMemoryContextByID(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string, contextID string) error {
	stmt := spanner.Statement{
		SQL: `DELETE FROM MemoryContexts WHERE UserId=@userId AND VaultId=@vaultId AND MemoryId=@memoryId AND ContextId=@contextId`,
		Params: map[string]interface{}{
			"userId":    userID,
			"vaultId":   vaultID.String(),
			"memoryId":  memoryID,
			"contextId": contextID,
		},
	}
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}

// convertJSONToMap attempts to coerce a value returned by Spanner's JSON column
// into map[string]interface{}. It is tolerant to the three shapes we may receive:
//  1. already a map[string]interface{}
//  2. string containing raw JSON (Spanner may return string when driver is mis-configured)
//  3. []byte with JSON payload
//
// Returns ok=false when the value cannot be parsed into an object.
func convertJSONToMap(v interface{}) (map[string]interface{}, error) {
	var obj map[string]interface{}
	switch val := v.(type) {
	case map[string]interface{}:
		return val, nil
	case string:
		if err := json.Unmarshal([]byte(val), &obj); err != nil {
			return nil, err
		}
		return obj, nil
	case []byte:
		if err := json.Unmarshal(val, &obj); err != nil {
			return nil, err
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("unsupported JSON type %T", v)
	}
}
