package localstate

import (
	"database/sql"
)

// EnsureSQLiteSchema creates core tables if they do not exist.
// This is a minimal subset of the full schema sufficient for local dev.
func EnsureSQLiteSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS Users (
            UserId TEXT PRIMARY KEY,
            Email TEXT NOT NULL UNIQUE,
            DisplayName TEXT,
            TimeZone TEXT NOT NULL,
            Status TEXT NOT NULL,
            CreationTime TIMESTAMP NOT NULL,
            LastActiveTime TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS Vaults (
            UserId TEXT NOT NULL,
            VaultId TEXT NOT NULL,
            Title TEXT NOT NULL,
            Description TEXT,
            CreationTime TIMESTAMP NOT NULL,
            PRIMARY KEY(UserId, VaultId),
            UNIQUE(UserId, Title)
        );`,
		`CREATE TABLE IF NOT EXISTS Memories (
            UserId TEXT NOT NULL,
            VaultId TEXT NOT NULL,
            MemoryId TEXT NOT NULL,
            MemoryType TEXT NOT NULL,
            Title TEXT NOT NULL,
            Description TEXT,
            CreationTime TIMESTAMP NOT NULL,
            Indexed BOOLEAN NOT NULL DEFAULT 0,
            PRIMARY KEY(UserId, VaultId, MemoryId),
            UNIQUE(VaultId, Title)
        );`,
		`CREATE TABLE IF NOT EXISTS MemoryEntries (
            UserId TEXT NOT NULL,
            VaultId TEXT NOT NULL,
            MemoryId TEXT NOT NULL,
            Title TEXT NOT NULL,
            CreationTime TIMESTAMP NOT NULL,
            EntryId TEXT NOT NULL,
            RawEntry TEXT NOT NULL,
            Summary TEXT,
            Metadata TEXT,
            Tags TEXT,
            CorrectionTime TIMESTAMP,
            CorrectedEntryMemoryId TEXT,
            CorrectedEntryCreationTime TIMESTAMP,
            CorrectionReason TEXT,
            LastUpdateTime TIMESTAMP,
            ExpirationTime TIMESTAMP,
            Indexed BOOLEAN NOT NULL DEFAULT 0,
            PRIMARY KEY(UserId, VaultId, MemoryId, CreationTime, EntryId)
        );`,
		`CREATE UNIQUE INDEX IF NOT EXISTS MemoryEntries_EntryId_Idx ON MemoryEntries(EntryId);`,
		`CREATE TABLE IF NOT EXISTS MemoryContexts (
            UserId TEXT NOT NULL,
            VaultId TEXT NOT NULL,
            MemoryId TEXT NOT NULL,
            Title TEXT NOT NULL,
            ContextId TEXT NOT NULL,
            Context TEXT NOT NULL,
            CreationTime TIMESTAMP NOT NULL,
            PRIMARY KEY(UserId, VaultId, MemoryId, ContextId)
        );`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
