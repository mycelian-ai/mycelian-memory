# Simple Actor Authorization System

This package implements a minimalist API key-based actor authorization for the Mycelian Memory system. No middleware, no complex context keys - just simple function calls.

## Overview

The system supports two deployment modes:

1. **SaaS Mode (Production)**: API keys resolved to actors via external authentication provider
2. **Local Mode (Development)**: Hardcoded API key resolved to local dev actor via MockAuthorizer

## Design Principles

- **No Middleware**: Handlers extract API keys directly from requests
- **Single Function**: One `Authorize()` call does authentication + authorization
- **No Context Keys**: No complex context propagation or extraction functions
- **Explicit**: Every handler explicitly calls authorization - can't forget it

## Components

### Authorizer Interface

```go
type Authorizer interface {
    ResolveActor(ctx context.Context, apiKey string) (*ActorInfo, error)
    CheckProjectPermission(ctx context.Context, actorID, projectID, permission string) (bool, error)
    ListActorProjects(ctx context.Context, actorID string) ([]string, error)
}

type ActorInfo struct {
    ActorID     string   `json:"actor_id"`     // Same as key_id
    ProjectID   string   `json:"project_id"`   // Which project this actor belongs to
    OrgID       string   `json:"org_id"`       // Which organization owns the project
    KeyType     string   `json:"key_type"`     // 'standard', 'admin'
    KeyName     string   `json:"key_name"`     // Human-readable name
    Permissions []string `json:"permissions"`  // Project-level permissions
}
```

### Implementations

#### MockAuthorizer (Local Mode)
- Recognizes hardcoded API key: `sk_local_mycelian_dev_key`
- Resolves to "mycelian-dev" actor with admin permissions
- Used for local development only (not beta, gamma, or prod stacks)

#### ProductionAuthorizer (SaaS Mode)
- Validates API keys against external authentication provider
- Resolves to real actor information from actor database
- TODO: Implementation pending

### Factory

```go
factory := NewAuthorizerFactory(cfg)
authorizer := factory.CreateAuthorizer()
```

### Handler Pattern (No Middleware)

```go
// Each handler does its own authorization
func CreateVaultHandler(w http.ResponseWriter, r *http.Request) {
    // Extract API key from request
    apiKey, err := auth.ExtractAPIKey(r)
    if err != nil {
        http.Error(w, "Unauthorized: "+err.Error(), 401)
        return
    }

    // Authorize in one call
    actorInfo, err := authorizer.Authorize(r.Context(), apiKey, "vault.create", "default")
    if err != nil {
        http.Error(w, "Unauthorized: "+err.Error(), 401)
        return
    }

    // Use actorInfo.ActorID for business logic
}
```

## Configuration

### Environment Variables

- `MEMORY_SERVER_DEV_MODE=true`: Enable development mode (MockAuthorizer)
- `MEMORY_SERVER_DEV_MODE=false` or unset: Production mode (ProductionAuthorizer)

### Client Configuration

#### Local Development
```go
// Use convenience constructor with hardcoded API key
client := mycelian.NewWithDevMode("http://localhost:8080")

// Or explicitly provide the hardcoded API key
client := mycelian.New("http://localhost:8080", "sk_local_mycelian_dev_key")
```

#### Production
```go
// Use real API key obtained from authentication provider
client := mycelian.New("https://api.mycelian.com", "sk_act_real_api_key_123")
```

## Usage Examples

### SaaS Deployment

```go
// Server side - no middleware, just authorizer
factory := NewAuthorizerFactory(cfg) // Production mode
authorizer := factory.CreateAuthorizer() // Returns ProductionAuthorizer

// In handlers:
apiKey, _ := auth.ExtractAPIKey(r)
actorInfo, _ := authorizer.Authorize(ctx, apiKey, "vault.create", "default")

// Client side
client := mycelian.New("https://api.mycelian.com", "sk_act_real_api_key")
```

### Local Development

```go
// Server side - no middleware, just authorizer
factory := NewAuthorizerFactory(cfg) // Development mode
authorizer := factory.CreateAuthorizer() // Returns MockAuthorizer

// In handlers:
apiKey, _ := auth.ExtractAPIKey(r)
actorInfo, _ := authorizer.Authorize(ctx, apiKey, "vault.create", "default")

// Client side
client := mycelian.NewWithDevMode("http://localhost:8080")
// Automatically uses "sk_local_mycelian_dev_key"
```

### Docker Development

```bash
# Server
docker run -e MEMORY_SERVER_DEV_MODE=true mycelian-memory

# Client uses hardcoded API key - no environment setup needed
```

## Security Features

- **API Key Authentication**: All requests must provide valid Bearer token
- **Provider Agnostic**: Works with any authentication provider via Authorizer interface
- **Local Dev Isolation**: MockAuthorizer only works in dev mode, never in production
- **Actor-Based Permissions**: Fine-grained access control via actor permissions
- **Audit Logging**: All authentication attempts logged for monitoring

## Error Handling

- `ErrInvalidAPIKey`: API key validation failed
- `ErrMissingAPIKey`: Authorization header missing
- `ErrActorNotFound`: Actor resolution failed
- `ErrPermissionDenied`: Actor lacks required permissions

## Requirements Satisfied

This implementation satisfies the Mycelian scoping model requirements:

- **Actor Model**: API keys are actors with unique identities
- **Provider Agnostic**: Authorizer interface supports any authentication provider
- **Local Development**: Hardcoded API key for local dev (security-conscious)
- **Project Isolation**: Actors belong to projects with controlled access
