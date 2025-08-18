package auth

import (
	"github.com/mycelian/mycelian-memory/server/internal/config"
)

// AuthorizerFactory creates the appropriate Authorizer based on environment
type AuthorizerFactory struct {
	config *config.Config
}

// NewAuthorizerFactory creates a new AuthorizerFactory
func NewAuthorizerFactory(cfg *config.Config) *AuthorizerFactory {
	return &AuthorizerFactory{
		config: cfg,
	}
}

// CreateAuthorizer creates the appropriate Authorizer based on development mode
func (f *AuthorizerFactory) CreateAuthorizer() Authorizer {
	if f.config.IsDevMode() {
		// Development mode: use mock authorizer with hardcoded API key
		return NewMockAuthorizer()
	}

	// Production mode: use real authorizer
	// TODO: implement ProductionAuthorizer that validates against real auth provider
	// For now, return mock authorizer - this will be replaced with real implementation
	return NewMockAuthorizer()
}

// IsDevMode returns true if development mode is enabled
func (f *AuthorizerFactory) IsDevMode() bool {
	return f.config.IsDevMode()
}
