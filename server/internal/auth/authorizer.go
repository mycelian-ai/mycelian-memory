package auth

import (
	"context"
)

// ActorInfo contains information about an authenticated actor
type ActorInfo struct {
	ActorID     string   `json:"actor_id"`    // Same as key_id
	ProjectID   string   `json:"project_id"`  // Which project this actor belongs to
	OrgID       string   `json:"org_id"`      // Which organization owns the project
	KeyType     string   `json:"key_type"`    // 'standard', 'admin'
	KeyName     string   `json:"key_name"`    // Human-readable name
	Permissions []string `json:"permissions"` // Project-level permissions
}

// Authorizer validates API keys and checks permissions in one call
type Authorizer interface {
	// Authorize validates API key and checks if actor can perform operation
	// Returns ActorInfo if authorized, error if authentication or authorization fails
	Authorize(ctx context.Context, apiKey, operation, resource string) (*ActorInfo, error)
}
