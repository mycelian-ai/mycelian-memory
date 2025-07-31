# Memory Backend REST API Documentation

> This is the **canonical** specification for the Memory Backend's current REST surface. It supersedes previous delta documents; all endpoints, request/response schemas, and examples are consolidated here.

## Table of Contents

1. [Health Check API](#health-check-api)
2. [User API](#user-api)
3. [Vault API](#vault-api)
4. [Memory API](#memory-api)
5. [Memory Entry API](#memory-entry-api)
6. [Context API](#context-api)
7. [Search API](#search-api)
8. [Error Response Format](#error-response-format)

---

## Health Check API

### Health Check
```http
GET /api/health
```

Checks the overall health of the service.

**Response (200) – Success**
```json
{
  "status": "healthy",
  "timestamp": "2025-07-26T19:58:00Z"
}
```

### Database Health Check
```http
GET /api/health/db
```

Checks the health of the database connection.

**Response (200) – Success**
```json
{
  "status": "healthy",
  "timestamp": "2025-07-26T19:58:00Z"
}
```

---

## User API

### Create User
```http
POST /api/users
Content-Type: application/json
```

Creates a new user. **User ID IS REQUIRED and is caller-supplied** (`userId`) and must:

* be 1-20 characters
* contain only lowercase letters (`a-z`), digits (`0-9`) and underscore (`_`)

**Request Body**
```json
{
  "userId": "local_user",             // required ‑ lowercase, digits, _
  "email": "user@example.com",          // required – valid e-mail
  "displayName": "John Doe",            // optional
  "timeZone": "America/New_York"        // optional (defaults UTC)
}
```

**Response (201) – Created**
```json
{
  "userId": "local_user",               // equals provided userId
  "email": "user@example.com",
  "displayName": "John Doe",
  "timeZone": "America/New_York",
  "status": "active",
  "creationTime": "2025-07-26T19:58:00Z",
  "lastActiveTime": null
}
```

**Response (409) – Conflict**  – userId already exists
```json
{
  "error": "Conflict",
  "code": 409,
  "message": "userId already exists"
}
```

### Get User
```http
GET /api/users/{userId}
```

Retrieves a user by their ID (the caller-supplied `userId`).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId |

_**Note:** For all endpoints below, `userId` is the caller-supplied **userId** (lowercase letters, digits, underscore, 1-20 chars).userIds are case-insensitive; the service stores them in lowercase and returns **409 Conflict** if a duplicate is requested.userIds cannot be changed after creation._

**Response (200) – Success**
```json
{
  "userId": "local_user",
  "email": "user@example.com",
  "displayName": "John Doe",
  "timeZone": "America/New_York",
  "status": "active",
  "creationTime": "2025-07-26T19:58:00Z",
  "lastActiveTime": "2025-07-26T20:00:00Z"
}
```

---

## Vault API

### Create Vault
```http
POST /api/users/{userId}/vaults
Content-Type: application/json
```

Creates a new vault for organizing memories.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |

**Request Body**
```json
{
  "title": "Work Projects",                           // required, max 50 chars, letters/digits/hyphen only
  "description": "Memories related to work projects"  // optional, max 500 chars
}
```

**Response (201) – Created**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "title": "Work Projects",
  "description": "Memories related to work projects",
  "creationTime": "2025-07-26T19:58:00Z"
}
```

### List Vaults
```http
GET /api/users/{userId}/vaults
```

Lists all vaults for a user.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |

**Response (200) – Success**
```json
{
  "vaults": [
    {
      "userId": "local_user",
      "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
      "title": "Work Projects",
      "description": "Memories related to work projects",
      "creationTime": "2025-07-26T19:58:00Z"
    }
  ],
  "count": 1
}
```

### Get Vault
```http
GET /api/users/{userId}/vaults/{vaultId}
```

Retrieves a specific vault.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |

**Response (200) – Success**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "title": "Work Projects",
  "description": "Memories related to work projects",
  "creationTime": "2025-07-26T19:58:00Z"
}
```

### Delete Vault
```http
DELETE /api/users/{userId}/vaults/{vaultId}
```

Deletes a vault. The vault must be empty (no memories).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |

**Response (204) – No Content**

---

## Memory API

### Create Memory
```http
POST /api/users/{userId}/vaults/{vaultId}/memories
Content-Type: application/json
```

Creates a new memory within a vault.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |

**Request Body**
```json
{
  "memoryType": "conversation",                       // required
  "title": "Project Planning Meeting",                // required, max 50 chars, letters/digits/hyphen only
  "description": "Q4 planning discussion with team"  // optional, max 500 chars
}
```

**Response (201) – Created**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
  "memoryType": "conversation",
  "title": "Project Planning Meeting",
  "description": "Q4 planning discussion with team",
  "creationTime": "2025-07-26T19:58:00Z"
}
```

### List Memories
```http
GET /api/users/{userId}/vaults/{vaultId}/memories
```

Lists all memories within a vault.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |

**Response (200) – Success**
```json
{
  "memories": [
    {
      "userId": "local_user",
      "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
      "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
      "memoryType": "conversation",
      "title": "Project Planning Meeting",
      "description": "Q4 planning discussion with team",
      "creationTime": "2025-07-26T19:58:00Z"
    }
  ],
  "count": 1
}
```

### Get Memory
```http
GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}
```

Retrieves a specific memory.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |
| `memoryId` | UUID | Memory identifier |

**Response (200) – Success**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
  "memoryType": "conversation",
  "title": "Project Planning Meeting",
  "description": "Q4 planning discussion with team",
  "creationTime": "2025-07-26T19:58:00Z"
}
```

### Delete Memory
```http
DELETE /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}
```

Deletes a memory and all its entries.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |
| `memoryId` | UUID | Memory identifier |

**Response (204) – No Content**

### Get Memory by Title
```http
GET /api/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}
```

Retrieves a memory by vault title and memory title (convenience endpoint).

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultTitle` | string | Vault title (must not be a UUID) |
| `memoryTitle` | string | Memory title |

**Response (200) – Success**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
  "memoryType": "conversation",
  "title": "Project Planning Meeting",
  "description": "Q4 planning discussion with team",
  "creationTime": "2025-07-26T19:58:00Z"
}
```

### List Memories by Vault Title
```http
GET /api/users/{userId}/vaults/{vaultTitle}/memories
```

Lists all memories within a vault identified by title.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultTitle` | string | Vault title (must not be a UUID) |

**Response (200) – Success**
```json
{
  "memories": [
    {
      "userId": "local_user",
      "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
      "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
      "memoryType": "conversation",
      "title": "Project Planning Meeting",
      "description": "Q4 planning discussion with team",
      "creationTime": "2025-07-26T19:58:00Z"
    }
  ],
  "count": 1
}
```

---

## Memory Entry API

### Create Memory Entry
```http
POST /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries
Content-Type: application/json
```

Creates a new entry in a memory.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |
| `memoryId` | UUID | Memory identifier |

**Request Body**
```json
{
  "rawEntry": "Discussed the new Kubernetes deployment strategy...",  // required, max 9000 chars
  "summary": "Kubernetes deployment planning",  // optional
  "metadata": {                                 // optional
    "participants": ["alice", "bob"],
    "location": "Conference Room A"
  },
  "tags": {                                     // optional
    "project": "infrastructure",
    "priority": "high"
  },
  "expirationTime": "2025-12-31T23:59:59Z"     // optional
}
```

**Field Constraints**
| Field | Type | Required | Max Length | Description |
|-------|------|----------|------------|-------------|
| `rawEntry` | string | Yes | 9000 chars | The main content of the memory entry |
| `summary` | string | No | - | Brief summary of the entry content |
| `metadata` | object | No | - | Structured metadata as JSON object |
| `tags` | object | No | - | Operational tags as JSON object |
| `expirationTime` | RFC3339 | No | - | When the entry should expire |

**Response (201) – Created**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
  "entryId": "entry-123e4567-e89b-12d3-a456-426614174003",
  "creationTime": "2025-07-26T19:58:00Z",
  "rawEntry": "Discussed the new Kubernetes deployment strategy...",
  "summary": "Kubernetes deployment planning",
  "metadata": {
    "participants": ["alice", "bob"],
    "location": "Conference Room A"
  },
  "tags": {
    "project": "infrastructure",
    "priority": "high"
  },
  "expirationTime": "2025-12-31T23:59:59Z",
  "correctionTime": null,
  "correctedEntryMemoryId": null,
  "correctedEntryCreationTime": null,
  "correctionReason": null,
  "lastUpdateTime": null,
  "deletionScheduledTime": null
}
```

### List Memory Entries
```http
GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries
```

Lists entries in a memory with optional pagination.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |
| `memoryId` | UUID | Memory identifier |

**Query Parameters**
| Name | Type | Description |
|------|------|-------------|
| `limit` | integer | Maximum entries to return (default: 100) |
| `before` | RFC3339 | Return entries before this timestamp |
| `after` | RFC3339 | Return entries after this timestamp |

**Response (200) – Success**
```json
{
  "entries": [
    {
      "userId": "local_user",
      "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
      "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
      "entryId": "entry-123e4567-e89b-12d3-a456-426614174003",
      "creationTime": "2025-07-26T19:58:00Z",
      "rawEntry": "Discussed the new Kubernetes deployment strategy...",
      "summary": "Kubernetes deployment planning",
      "metadata": {
        "participants": ["alice", "bob"],
        "location": "Conference Room A"
      },
      "tags": {
        "project": "infrastructure",
        "priority": "high"
      },
      "expirationTime": "2025-12-31T23:59:59Z",
      "correctionTime": null,
      "correctedEntryMemoryId": null,
      "correctedEntryCreationTime": null,
      "correctionReason": null,
      "lastUpdateTime": null,
      "deletionScheduledTime": null
    }
  ],
  "count": 1
}
```

### Update Memory Entry Tags
```http
PATCH /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{creationTime}/tags
Content-Type: application/json
```

Updates the tags of a memory entry.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |
| `memoryId` | UUID | Memory identifier |
| `creationTime` | RFC3339/RFC3339Nano | Entry creation timestamp |

**Request Body**
```json
{
  "tags": {
    "project": "infrastructure",
    "priority": "critical",
    "reviewed": true
  }
}
```

**Response (200) – Success**
Returns the updated memory entry with the same structure as Create Memory Entry response.

---

## Context API

### Put Context Snapshot
```http
PUT /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts
Content-Type: application/json
```

Creates or updates the latest context snapshot for a memory. This is idempotent – issuing the call multiple times with the same payload overwrites the previous snapshot.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |
| `memoryId` | UUID | Memory identifier |

**Request Body**
```json
{
  "context": {
    "agenda": "Design review at 2 PM",
    "dri": "alice@example.com",
    "currentPhase": "implementation",
    "nextSteps": ["code review", "deployment planning"]
  }
}
```

Rules:
* Payload must be a **non-empty JSON object** with string keys
* String values must be non-empty; nested objects and arrays allowed
* Max size 256 KiB
* A new memory starts with default context until overwritten

**Response (201) – Created**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
  "contextId": "ctx-789a0123-e89b-12d3-a456-426614174003",
  "context": {
    "agenda": "Design review at 2 PM",
    "dri": "alice@example.com",
    "currentPhase": "implementation",
    "nextSteps": ["code review", "deployment planning"]
  },
  "creationTime": "2025-07-26T19:58:00Z"
}
```

### Get Latest Context Snapshot
```http
GET /api/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts
```

Returns the most recent context snapshot for the memory. A brand-new memory always has a default context, so this endpoint never returns 404 once the memory exists.

**Path Parameters**
| Name | Type | Description |
|------|------|-------------|
| `userId` | string |userId (caller-supplied) |
| `vaultId` | UUID | Vault identifier |
| `memoryId` | UUID | Memory identifier |

**Response (200) – Success**
```json
{
  "userId": "local_user",
  "vaultId": "vault-456e7890-e89b-12d3-a456-426614174001",
  "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
  "contextId": "ctx-789a0123-e89b-12d3-a456-426614174003",
  "context": {
    "agenda": "Design review at 2 PM",
    "dri": "alice@example.com",
    "currentPhase": "implementation",
    "nextSteps": ["code review", "deployment planning"]
  },
  "creationTime": "2025-07-26T19:58:00Z"
}
```

---

## Search API

### Search
```http
POST /api/search
Content-Type: application/json
```

Performs hybrid semantic/keyword search within a specific memory scoped to the authenticated user (multi-tenant isolation in Weaviate).

**Request Body**
```json
{
  "userId": "local_user",
  "memoryId": "mem-789a0123-e89b-12d3-a456-426614174002",
  "query": "What did I say about kubernetes yesterday?",
  "topK": 10          // optional (1-100, default 10)
}
```

**Response (200) – Success**
```json
{
  "entries": [
    {
      "entryId": "entry-123e4567-e89b-12d3-a456-426614174002",
      "userId": "local_user",
      "memoryId": "mem-456e7890-e89b-12d3-a456-426614174001",
      "summary": "Discussed Waviate search integration",
      "rawEntry": "Yesterday we talked about integrating Waviate hybrid search...",
      "score": 0.91
    }
  ],
  "count": 1,
  "latestContext": "Reminder to review yesterday's integration notes",
  "contextTimestamp": "2025-06-30T14:12:01Z",
  "bestContext": "Context snapshot closely matching the query",
  "bestContextTimestamp": "2025-06-30T11:05:47Z",
  "bestContextScore": 0.83
}
```

Field descriptions:
- `entries`: Top-K matching `MemoryEntry` records (subset of fields shown)
- `count`: Number of entries returned (≤ `topK`)
- `latestContext`: The most recent context snapshot for the memory. Present whenever a snapshot exists, even if the search returns zero entries
- `contextTimestamp`: RFC3339 timestamp indicating when `latestContext` was captured
- `bestContext`: Context snapshot returned by hybrid search that best matches the query. May be omitted when no match found
- `bestContextTimestamp`: Timestamp of `bestContext` snapshot
- `bestContextScore`: Similarity score (0-1) of the `bestContext` match

---

## Error Response Format

All endpoints use a consistent error response format:

**Validation Error (400)**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": {
      "field": "email",
      "reason": "Invalid email format"
    }
  }
}
```

**Not Found Error (404)**
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found",
    "details": {
      "resource": "memory",
      "id": "mem-789a0123-e89b-12d3-a456-426614174002"
    }
  }
}
```

**Internal Server Error (500)**
```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "An internal error occurred",
    "details": null
  }
}
```

---

## Notes

### Authentication & Authorization
- All endpoints require authentication (implementation pending)
- Users can only access their own resources
- Multi-tenant isolation enforced at storage and search layers

### ID Generation
* **User IDs**: caller-supplieduserIds
* **Vault / Memory / Entry / Context IDs**: client-generated UUIDv4, unchanged.
* Rationale remains in ADR-0006;userIds improve DX for local / scripted environments.

### Timestamps
- All timestamps use RFC3339 format
- Creation timestamps are server-generated
- Timezone information preserved where applicable

### Title Validation Rules
- **Vault and Memory titles** have the following constraints:
  - Maximum length: 50 characters
  - Allowed characters: ASCII letters (A-Z, a-z), digits (0-9), and hyphen (-)
  - Required field (cannot be empty)
  - Must match regex pattern: `^[A-Za-z0-9\-]+$`

### Pagination
- List endpoints support cursor-based pagination using `before`/`after` timestamps
- Default and maximum limits vary by endpoint
- Results ordered by creation time (newest first)

### Soft Deletion
- Deleted resources are marked for deletion but retained for 30 days
- See ADR-0007 for retention policy details
