# API Reference

## Overview

The Memory Service provides a REST API for managing users, vaults, memories, entries, and contexts. All endpoints are versioned under `/v0`. Most endpoints respond with JSON. Context endpoints use plain text for request/response bodies.

**Base URL**: `http://localhost:11545/v0`

## Authentication

Currently, the API operates without authentication. All requests are processed directly.

## Response Format

### Success Responses
- **200 OK**: Request successful, data returned
- **201 Created**: Resource created successfully
- **204 No Content**: Request successful, no data returned

### Error Responses
- **400 Bad Request**: Invalid request parameters
- **404 Not Found**: Resource not found
- **500 Internal Server Error**: Server error

## Health Check

### Check Service Health
```
GET /v0/health
```

Returns the current health status of the service and its dependencies.

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-01T12:00:00Z"
}
```

Status values: `healthy` or `unhealthy`

## Users

### Create User
```
POST /v0/users
```

**Request Body**:
```json
{
  "userId": "string",
  "email": "string",
  "displayName": "string",
  "timeZone": "string"
}
```

**Response**: `201 Created`
```json
{
  "userId": "user123",
  "email": "user@example.com",
  "displayName": "User Name",
  "timeZone": "UTC",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z"
}
```

### Get User
```
GET /v0/users/{userId}
```

**Parameters**:
- `userId` (path): User identifier

**Response**: `200 OK`
```json
{
  "userId": "user123",
  "email": "user@example.com",
  "displayName": "User Name",
  "timeZone": "UTC",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z"
}
```

## Vaults

### Create Vault
```
POST /v0/users/{userId}/vaults
```

**Parameters**:
- `userId` (path): User identifier

**Request Body**:
```json
{
  "title": "string",
  "description": "string"
}
```

**Response**: `201 Created`
```json
{
  "vaultId": "vault123",
  "userId": "user123",
  "title": "Vault Title",
  "description": "Vault description",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z"
}
```

### List Vaults
```
GET /v0/users/{userId}/vaults
```

**Parameters**:
- `userId` (path): User identifier

**Response**: `200 OK`
```json
{
  "vaults": [
    {
      "vaultId": "vault123",
      "userId": "user123",
      "title": "Vault Title",
      "description": "Vault description",
      "created_at": "2025-01-01T12:00:00Z",
      "updated_at": "2025-01-01T12:00:00Z"
    }
  ],
  "count": 1
}
```

### Get Vault
```
GET /v0/users/{userId}/vaults/{vaultId}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier

**Response**: `200 OK`
```json
{
  "vaultId": "vault123",
  "userId": "user123",
  "title": "Vault Title",
  "description": "Vault description",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z"
}
```

### Delete Vault
```
DELETE /v0/users/{userId}/vaults/{vaultId}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier

**Response**: `204 No Content`

### Attach Memory to Vault
```
POST /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/attach
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier

**Response**: `201 Created`

## Memories

### Create Memory
```
POST /v0/users/{userId}/vaults/{vaultId}/memories
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier

**Request Body**:
```json
{
  "title": "string",
  "memoryType": "string",
  "description": "string"
}
```

**Response**: `201 Created`
```json
{
  "memoryId": "memory123",
  "vaultId": "vault123",
  "userId": "user123",
  "title": "Memory Title",
  "memoryType": "conversation",
  "description": "Memory description",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z"
}
```

### List Memories
```
GET /v0/users/{userId}/vaults/{vaultId}/memories
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier

**Response**: `200 OK`
```json
{
  "memories": [
    {
      "memoryId": "memory123",
      "vaultId": "vault123",
      "userId": "user123",
      "title": "Memory Title",
      "memoryType": "conversation",
      "description": "Memory description",
      "created_at": "2025-01-01T12:00:00Z",
      "updated_at": "2025-01-01T12:00:00Z"
    }
  ],
  "count": 1
}
```

### Get Memory
```
GET /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier

**Response**: `200 OK`
```json
{
  "memoryId": "memory123",
  "vaultId": "vault123",
  "userId": "user123",
  "title": "Memory Title",
  "memoryType": "conversation",
  "description": "Memory description",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z"
}
```

### Delete Memory
```
DELETE /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier

**Response**: `204 No Content`

### List Memories by Vault Title
```
GET /v0/users/{userId}/vaults/{vaultTitle}/memories
```

**Parameters**:
- `userId` (path): User identifier
- `vaultTitle` (path): Vault title (URL encoded)

**Response**: `200 OK` (same format as List Memories)

### Get Memory by Title
```
GET /v0/users/{userId}/vaults/{vaultTitle}/memories/{memoryTitle}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultTitle` (path): Vault title (URL encoded)
- `memoryTitle` (path): Memory title (URL encoded)

**Response**: `200 OK` (same format as Get Memory)

## Entries

### List Memory Entries
```
GET /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier

**Query Parameters**:
- `limit` (optional): Maximum number of entries to return
- `offset` (optional): Number of entries to skip

**Response**: `200 OK`
```json
{
  "entries": [
    {
      "entryId": "entry123",
      "userId": "user123",
      "vaultId": "vault123",
      "memoryId": "memory123",
      "rawEntry": "Entry content",
      "tags": ["tag1", "tag2"],
      "creationTime": "2025-01-01T12:00:00Z"
    }
  ],
  "count": 1
}
```

### Create Memory Entry
```
POST /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier

**Request Body**:
```json
{
  "rawEntry": "string",
  "tags": ["string"]
}
```

**Response**: `201 Created`
```json
{
  "entryId": "entry123",
  "userId": "user123",
  "vaultId": "vault123",
  "memoryId": "memory123",
  "rawEntry": "Entry content",
  "tags": ["tag1", "tag2"],
  "creationTime": "2025-01-01T12:00:00Z"
}
```

### Get Memory Entry
```
GET /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier
- `entryId` (path): Entry identifier

**Response**: `200 OK`
```json
{
  "entryId": "entry123",
  "userId": "user123",
  "vaultId": "vault123",
  "memoryId": "memory123",
  "rawEntry": "Entry content",
  "tags": ["tag1", "tag2"],
  "creationTime": "2025-01-01T12:00:00Z"
}
```

### Delete Memory Entry
```
DELETE /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier
- `entryId` (path): Entry identifier

**Response**: `204 No Content`

### Update Memory Entry Tags
```
PATCH /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/entries/{entryId}/tags
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier
- `entryId` (path): Entry identifier

**Request Body**:
```json
{
  "tags": ["string"]
}
```

**Response**: `200 OK`

## Contexts

### Put Memory Context
```
PUT /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts
```

**Headers**:
- `Content-Type: text/plain; charset=utf-8`

**Body**:
- Raw text (entire context document), UTF‑8.

**Validation**:
- Valid UTF‑8 only
- Allowed control chars: TAB, LF, CR
- Rejects other control chars and Unicode noncharacters
- Max length limited by characters via `MEMORY_SERVER_MAX_CONTEXT_CHARS` (default 65536)

**Response**: `201 Created`

### Get Latest Memory Context
```
GET /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts
```

**Headers**:
- `Accept: text/plain`

**Response**: `200 OK`
- Body is raw text of the latest context document (`text/plain; charset=utf-8`).

### Delete Memory Context
```
DELETE /v0/users/{userId}/vaults/{vaultId}/memories/{memoryId}/contexts/{contextId}
```

**Parameters**:
- `userId` (path): User identifier
- `vaultId` (path): Vault identifier
- `memoryId` (path): Memory identifier
- `contextId` (path): Context identifier

**Response**: `204 No Content`

## Search

### Search Memories
```
POST /v0/search
```

Performs hybrid semantic and keyword search within a memory.

**Request Body**:
```json
{
  "memoryId": "string",    // Required: Target memory UUID
  "query": "string",        // Required: Search query text
  "top_ke": 5,             // Optional: Number of entries to return (default: 5, range: 0-10)
  "top_kc": 2              // Optional: Number of context shards to return (default: 2, range: 1-3)
}
```

**Response**: `200 OK`
```json
{
  "entries": [
    {
      "entryId": "entry123",
      "actorId": "user123",
      "memoryId": "memory123",
      "summary": "Entry summary",
      "rawEntry": "Full entry content",
      "score": 0.95,
      "creationTime": "2025-01-01T12:00:00Z"
    }
  ],
  "count": 1,
  "latestContext": "Latest context text...",
  "latestContextTimestamp": "2025-01-01T12:30:00Z",
  "contexts": [
    {
      "context": "Context shard text...",
      "timestamp": "2025-01-01T11:00:00Z",
      "score": 0.92
    }
  ]
}
```

**Notes**:
- `top_ke=0` returns no entries (useful for context-only searches)
- Timestamps enable understanding temporal evolution
- Contexts are sorted by relevance score (descending)
- Authorization via Bearer token in header

## Data Types

### User
- `userId`: String, unique identifier
- `email`: String, user email address
- `displayName`: String, user display name
- `timeZone`: String, timezone identifier (e.g., "UTC", "America/New_York")
- `created_at`: ISO 8601 timestamp
- `updated_at`: ISO 8601 timestamp

### Vault
- `vaultId`: String, unique identifier
- `userId`: String, owner user identifier
- `title`: String, vault title
- `description`: String, vault description
- `created_at`: ISO 8601 timestamp
- `updated_at`: ISO 8601 timestamp

### Memory
- `memoryId`: String, unique identifier
- `vaultId`: String, parent vault identifier
- `userId`: String, owner user identifier
- `title`: String, memory title
- `memoryType`: String, type of memory (e.g., "conversation", "document")
- `description`: String, memory description
- `created_at`: ISO 8601 timestamp
- `updated_at`: ISO 8601 timestamp

### Entry
- `entryId`: String, unique identifier
- `userId`: String, owner user identifier
- `vaultId`: String, parent vault identifier
- `memoryId`: String, parent memory identifier
- `rawEntry`: String, entry content
- `tags`: Array of strings, entry tags
- `creationTime`: ISO 8601 timestamp

### Context
- `contextId`: String, unique identifier
- `memoryId`: String, parent memory identifier
- `context`: String (raw text document)
- `createdAt`: ISO 8601 timestamp

## Error Handling

All error responses follow this format:
```json
{
  "error": "string",
  "message": "string",
  "code": "string"
}
```

Common error scenarios:
- Invalid user ID format returns `400 Bad Request`
- Non-existent resources return `404 Not Found`
- Malformed JSON returns `400 Bad Request`
- Server errors return `500 Internal Server Error`

## Rate Limiting

No rate limiting is currently implemented.

## Versioning

The API is versioned using URL prefixes. Current version is `v0`. Future versions will increment (e.g., `v1`, `v2`).


## Migration Guide

### Search API Breaking Changes (v0)

**Old Parameters (Deprecated)**:
- `topK`: Legacy combined top-k parameter
- `ke`: Entries top-k
- `kc`: Context shards top-k

**New Parameters**:
- `top_ke`: Number of entries (default: 5, range: 0-10)
- `top_kc`: Number of context shards (default: 2, range: 1-3)

**Response Changes**:
- `contextTimestamp` → `latestContextTimestamp`
- `contexts` array is now always present (can be empty)
- Entries now include `creationTime` field

**Migration Example**:
```json
// Old request
{
  "memoryId": "m1",
  "query": "search text",
  "topK": 10,
  "ke": 5,
  "kc": 3
}

// New request
{
  "memoryId": "m1",
  "query": "search text",
  "top_ke": 5,
  "top_kc": 3
}
```
