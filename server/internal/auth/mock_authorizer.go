package auth

import (
	"context"
	"errors"
)

const (
	// LocalDevAPIKey is the hardcoded API key for local development only
	LocalDevAPIKey = "sk_local_mycelian_dev_key"
)

// MockAuthorizer provides a simple authorizer for local development
// It only recognizes the hardcoded LocalDevAPIKey and resolves it to a mycelian-dev actor
type MockAuthorizer struct{}

// NewMockAuthorizer creates a new MockAuthorizer for local development
func NewMockAuthorizer() *MockAuthorizer {
	return &MockAuthorizer{}
}

// Authorize validates the hardcoded API key and checks permissions in one call
func (m *MockAuthorizer) Authorize(ctx context.Context, apiKey, operation, resource string) (*ActorInfo, error) {
	if apiKey != LocalDevAPIKey {
		return nil, errors.New("invalid API key for local development")
	}

	// Local dev actor has admin access to everything
	return &ActorInfo{
		ActorID:     "mycelian-dev",
		ProjectID:   "local-dev-project",
		OrgID:       "local-dev-org",
		KeyType:     "admin",
		KeyName:     "Local Development Key",
		Permissions: []string{"*"}, // Wildcard for admin - can do anything
	}, nil
}
