-- Synapse Memory Product - Database Schema
-- Google Cloud Spanner (GoogleSQL Dialect)

-- User management for the Memory product
CREATE TABLE Users (
    UserId STRING(36) NOT NULL,              -- Client-generated UUID4
    Email STRING(320) NOT NULL,           -- RFC 5321 max email length
    DisplayName STRING(255),              -- Optional display name
    TimeZone STRING(50) NOT NULL,         -- For proper timestamp handling (default UTC in app)
    Status STRING(20) NOT NULL,           -- ACTIVE, SUSPENDED, DELETED (default ACTIVE in app)
    CreationTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    LastActiveTime TIMESTAMP OPTIONS (allow_commit_timestamp=true),  -- For inactive user cleanup
) PRIMARY KEY (UserId);

CREATE UNIQUE INDEX Users_Email_Idx ON Users(Email);

-- Vaults group memories for a user
CREATE TABLE Vaults (
    UserId STRING(36) NOT NULL,
    VaultId STRING(36) NOT NULL,           -- Client-generated UUID4
    Title STRING(50) NOT NULL,
    Description STRING(2048),              -- Optional free-form description
    CreationTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (UserId, VaultId),
INTERLEAVE IN PARENT Users ON DELETE CASCADE;

ALTER TABLE Vaults ADD CONSTRAINT FK_Vaults_Users FOREIGN KEY (UserId) REFERENCES Users(UserId);

-- Unique title per user
CREATE UNIQUE INDEX Vaults_Title_UQ ON Vaults(UserId, Title);

-- Memory instances for the Memory product
CREATE TABLE Memories (
    UserId STRING(36) NOT NULL,
    VaultId STRING(36) NOT NULL,             -- Parent vault
    MemoryId STRING(36) NOT NULL,            -- Client-generated UUID4
    MemoryType STRING(50) NOT NULL,       -- 'CONVERSATION', 'PROJECT', 'CONTEXT', etc.
    Title STRING(50) NOT NULL,
    Description STRING(2048),              -- Optional free-form description (≤2 KB)
    CreationTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (UserId, VaultId, MemoryId),
INTERLEAVE IN PARENT Vaults ON DELETE CASCADE;

ALTER TABLE Memories ADD CONSTRAINT FK_Memories_Vaults FOREIGN KEY (UserId, VaultId) REFERENCES Vaults(UserId, VaultId);

-- Unique title per vault
CREATE UNIQUE INDEX Memories_Title_UQ ON Memories(VaultId, Title);

-- Memory entries - Append-only log with corrections and explicit deletion allowed
CREATE TABLE MemoryEntries (
    UserId STRING(36) NOT NULL,
    VaultId STRING(36) NOT NULL,
    MemoryId STRING(36) NOT NULL,
    Title STRING(50),
    CreationTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true), -- Natural chronological ordering
    EntryId STRING(36) NOT NULL,             -- Client-generated UUID4 for external references
    RawEntry STRING(MAX) NOT NULL,        -- Text content only
    Summary STRING(512),                  -- AI-generated summary optimized for vector search (512 chars)
    Metadata JSON,                        -- Flexible key-value pairs (immutable)
    Tags JSON,                            -- Mutable operational metadata (status, workflow, etc.)
    CorrectionTime TIMESTAMP,             -- When this entry was corrected
    CorrectedEntryMemoryId STRING(36),    -- Points to memory containing correction entry
    CorrectedEntryCreationTime TIMESTAMP, -- Points to correction entry timestamp  
    CorrectionReason STRING(MAX),         -- Why this entry was corrected
    LastUpdateTime TIMESTAMP OPTIONS (allow_commit_timestamp=true), -- When summary was last updated
    ExpirationTime TIMESTAMP,             -- TTL support
    
    -- Ensure correction fields are all-or-nothing
    CONSTRAINT correction_consistency CHECK (
        (CorrectionTime IS NULL AND CorrectedEntryMemoryId IS NULL 
         AND CorrectedEntryCreationTime IS NULL AND CorrectionReason IS NULL)
        OR
        (CorrectionTime IS NOT NULL AND CorrectedEntryMemoryId IS NOT NULL 
         AND CorrectedEntryCreationTime IS NOT NULL AND CorrectionReason IS NOT NULL)
    )
    ) PRIMARY KEY (UserId, VaultId, MemoryId, CreationTime), -- Timestamp-based chronological ordering
INTERLEAVE IN PARENT Memories ON DELETE CASCADE; 

ALTER TABLE MemoryEntries ADD CONSTRAINT FK_MemoryEntries_Memories FOREIGN KEY (UserId, VaultId, MemoryId) REFERENCES Memories(UserId, VaultId, MemoryId);

-- Direct lookup by EntryId regardless of parent path
CREATE UNIQUE INDEX MemoryEntries_EntryId_Idx ON MemoryEntries(EntryId);

-- Memory contexts – Decoupled context snapshots per memory (append-only)
CREATE TABLE MemoryContexts (
    UserId STRING(36) NOT NULL,
    VaultId STRING(36) NOT NULL,
    MemoryId STRING(36) NOT NULL,
    Title STRING(50),
    ContextId STRING(36) NOT NULL,           -- Client-generated UUID4 for versioning
    Context JSON NOT NULL,                  -- Raw JSON context (≤256 KiB)
    CreationTime TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true)
) PRIMARY KEY (UserId, VaultId, MemoryId, ContextId),
INTERLEAVE IN PARENT Memories ON DELETE CASCADE;

ALTER TABLE MemoryContexts ADD CONSTRAINT FK_MemoryContexts_Memories FOREIGN KEY (UserId, VaultId, MemoryId) REFERENCES Memories(UserId, VaultId, MemoryId);

-- Quick access index for "latest context per memory"
CREATE INDEX MemoryContexts_CreationTime_Idx
ON MemoryContexts(UserId, VaultId, MemoryId, CreationTime DESC);
