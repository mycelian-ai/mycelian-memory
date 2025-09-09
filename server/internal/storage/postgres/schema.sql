-- PostgreSQL schema for Mycelian Memory (parity with ADR 0014)
-- Users table eliminated - actor_id is now treated as opaque string identifier

-- Vaults
CREATE TABLE IF NOT EXISTS vaults (
  actor_id       TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  title          TEXT NOT NULL,
  description    TEXT,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (actor_id, vault_id),
  UNIQUE (actor_id, title)
);

-- Memories
CREATE TABLE IF NOT EXISTS memories (
  actor_id       TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  memory_id      TEXT NOT NULL,
  memory_type    TEXT NOT NULL,
  title          TEXT NOT NULL,
  description    TEXT,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (actor_id, vault_id, memory_id),
  UNIQUE (vault_id, title)
);

-- MemoryEntries
CREATE TABLE IF NOT EXISTS memory_entries (
  actor_id       TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  memory_id      TEXT NOT NULL,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  entry_id       TEXT NOT NULL,
  raw_entry      TEXT NOT NULL,
  summary        TEXT,
  metadata       JSONB,
  tags           JSONB,
  correction_time TIMESTAMPTZ,
  corrected_entry_memory_id TEXT,
  corrected_entry_creation_time TIMESTAMPTZ,
  correction_reason TEXT,
  last_update_time TIMESTAMPTZ,
  PRIMARY KEY (actor_id, vault_id, memory_id, creation_time, entry_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS memory_entries_entry_id_uq ON memory_entries(entry_id);
CREATE INDEX IF NOT EXISTS memory_entries_recent_idx ON memory_entries(actor_id, vault_id, memory_id, creation_time DESC);

-- MemoryContexts
CREATE TABLE IF NOT EXISTS memory_contexts (
  actor_id       TEXT NOT NULL,
  vault_id       TEXT NOT NULL,
  memory_id      TEXT NOT NULL,
  context_id     TEXT NOT NULL,
  context        TEXT NOT NULL,
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (actor_id, vault_id, memory_id, context_id)
);

-- Outbox for Weaviate sync
CREATE TABLE IF NOT EXISTS outbox (
  id             BIGSERIAL PRIMARY KEY,
  aggregate_id   TEXT NOT NULL,
  op             TEXT NOT NULL,
  payload        JSONB NOT NULL,
  status         TEXT NOT NULL DEFAULT 'pending',
  attempt_count  INT NOT NULL DEFAULT 0,
  leased_until   TIMESTAMPTZ,
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  creation_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
  update_time    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS outbox_ready_idx ON outbox(status, next_attempt_at);
