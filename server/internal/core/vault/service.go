package vault

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service contains the core business logic for vault operations.
type Service struct{}

// NewService creates a new vault service.
func NewService() *Service { return &Service{} }

// CreateVault creates a new vault for a user.
func (s *Service) CreateVault(ctx context.Context, req CreateVaultRequest) (interface{}, error) {
	if err := s.validateCreateVaultRequest(req); err != nil {
		return nil, err
	}

	vaultID := uuid.New()

	log.Info().Str("userID", req.UserID).Str("vaultID", vaultID.String()).Msg("Creating vault")
	_ = strings.Contains // keep import
	return nil, nil
}

// GetVault retrieves a vault by ID.
func (s *Service) GetVault(ctx context.Context, userID string, vaultID uuid.UUID) (interface{}, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if vaultID == uuid.Nil {
		return nil, fmt.Errorf("vault ID is required")
	}
	return nil, fmt.Errorf("not implemented")
}

// GetVaultByTitle retrieves a vault by userID and unique title.
func (s *Service) GetVaultByTitle(ctx context.Context, userID string, title string) (interface{}, error) {
	if userID == "" || title == "" {
		return nil, fmt.Errorf("userID and title are required")
	}
	return nil, fmt.Errorf("not implemented")
}

// ListVaults lists all vaults for a user.
func (s *Service) ListVaults(ctx context.Context, userID string) (interface{}, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	return nil, fmt.Errorf("not implemented")
}

// DeleteVault deletes a vault. Underlying storage should cascade delete associations.
func (s *Service) DeleteVault(ctx context.Context, userID string, vaultID uuid.UUID) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if vaultID == uuid.Nil {
		return fmt.Errorf("vault ID is required")
	}
	log.Info().Str("userID", userID).Str("vaultID", vaultID.String()).Msg("Deleting vault")
	return fmt.Errorf("not implemented")
}

// AddMemoryToVault associates a memory with a vault.
func (s *Service) AddMemoryToVault(ctx context.Context, req AddMemoryToVaultRequest) error {
	if err := s.validateMemoryAssociationRequest(req.UserID, req.VaultID, req.MemoryID); err != nil {
		return err
	}
	return fmt.Errorf("not implemented")
}

// DeleteMemoryFromVault removes a memory association from a vault.
func (s *Service) DeleteMemoryFromVault(ctx context.Context, req DeleteMemoryFromVaultRequest) error {
	if err := s.validateMemoryAssociationRequest(req.UserID, req.VaultID, req.MemoryID); err != nil {
		return err
	}
	return fmt.Errorf("not implemented")
}

// Validation helpers
func (s *Service) validateCreateVaultRequest(req CreateVaultRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) > 50 {
		return fmt.Errorf("title exceeds 50 characters")
	}
	if !titleRx.MatchString(req.Title) {
		return fmt.Errorf("title contains invalid characters; allowed letters, digits, hyphen")
	}
	if req.Description != nil && len(*req.Description) > 500 {
		return fmt.Errorf("description exceeds 500 characters")
	}
	return nil
}

func (s *Service) validateMemoryAssociationRequest(userID string, vaultID uuid.UUID, memoryID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if vaultID == uuid.Nil {
		return fmt.Errorf("vault ID is required")
	}
	if memoryID == "" {
		return fmt.Errorf("memory ID is required")
	}
	return nil
}

// title validation regex shared by vault & memory services (ASCII letters, digits, space, hyphen, underscore, apostrophe)
var titleRx = regexp.MustCompile(`^[A-Za-z0-9\-]+$`)

// allowed lowercase letters, digits, hyphen, 1-50 chars
