package vault

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	memcore "memory-backend/internal/core/memory"
	"memory-backend/internal/storage"
)

// Service contains the core business logic for vault operations.
type Service struct {
	storage storage.Storage
}

// NewService creates a new vault service.
func NewService(storage storage.Storage) *Service {
	return &Service{storage: storage}
}

// CreateVault creates a new vault for a user.
func (s *Service) CreateVault(ctx context.Context, req CreateVaultRequest) (*storage.Vault, error) {
	if err := s.validateCreateVaultRequest(req); err != nil {
		return nil, err
	}

	vaultID := uuid.New()

	storageReq := storage.CreateVaultRequest{
		UserID:      req.UserID,
		VaultID:     vaultID,
		Title:       req.Title,
		Description: req.Description,
	}

	log.Info().Str("userID", req.UserID).Str("vaultID", vaultID.String()).Msg("Creating vault")

	v, err := s.storage.CreateVault(ctx, storageReq)
	if err != nil {
		// Detect foreign key violation indicating missing user row
		if strings.Contains(err.Error(), "parent table") || strings.Contains(err.Error(), "FK_Vaults_Users") || strings.Contains(err.Error(), "foreign key") {
			nf := memcore.NewNotFoundError("userID", "user not found")
			log.Error().Err(err).Str("userID", req.UserID).Str("vaultID", vaultID.String()).Msg("Failed to create vault: user does not exist")
			return nil, nf
		}
		log.Error().Err(err).Str("userID", req.UserID).Str("vaultID", vaultID.String()).Msg("Failed to create vault")
		return nil, err
	}
	return v, nil
}

// GetVault retrieves a vault by ID.
func (s *Service) GetVault(ctx context.Context, userID string, vaultID uuid.UUID) (*storage.Vault, error) {
	if userID == "" {
		return nil, memcore.NewValidationError("userID", "user ID is required")
	}
	if vaultID == uuid.Nil {
		return nil, memcore.NewValidationError("vaultID", "vault ID is required")
	}
	return s.storage.GetVault(ctx, userID, vaultID)
}

// GetVaultByTitle retrieves a vault by userID and unique title.
func (s *Service) GetVaultByTitle(ctx context.Context, userID string, title string) (*storage.Vault, error) {
	if userID == "" || title == "" {
		return nil, memcore.NewValidationError("title", "userID and title are required")
	}
	v, err := s.storage.GetVaultByTitle(ctx, userID, title)
	if err != nil {
		log.Warn().Str("userID", userID).Str("title", title).Err(err).Msg("GetVaultByTitle failed")
	}
	return v, err
}

// ListVaults lists all vaults for a user.
func (s *Service) ListVaults(ctx context.Context, userID string) ([]*storage.Vault, error) {
	if userID == "" {
		return nil, memcore.NewValidationError("userID", "user ID is required")
	}
	vts, err := s.storage.ListVaults(ctx, userID)
	if err != nil {
		log.Warn().Str("userID", userID).Err(err).Msg("ListVaults failed")
	}
	return vts, err
}

// DeleteVault deletes a vault. Underlying storage should cascade delete associations.
func (s *Service) DeleteVault(ctx context.Context, userID string, vaultID uuid.UUID) error {
	if userID == "" {
		return memcore.NewValidationError("userID", "user ID is required")
	}
	if vaultID == uuid.Nil {
		return memcore.NewValidationError("vaultID", "vault ID is required")
	}
	log.Info().Str("userID", userID).Str("vaultID", vaultID.String()).Msg("Deleting vault")
	return s.storage.DeleteVault(ctx, userID, vaultID)
}

// AddMemoryToVault associates a memory with a vault.
func (s *Service) AddMemoryToVault(ctx context.Context, req AddMemoryToVaultRequest) error {
	if err := s.validateMemoryAssociationRequest(req.UserID, req.VaultID, req.MemoryID); err != nil {
		return err
	}
	storageReq := storage.AddMemoryToVaultRequest(req)
	return s.storage.AddMemoryToVault(ctx, storageReq)
}

// DeleteMemoryFromVault removes a memory association from a vault.
func (s *Service) DeleteMemoryFromVault(ctx context.Context, req DeleteMemoryFromVaultRequest) error {
	if err := s.validateMemoryAssociationRequest(req.UserID, req.VaultID, req.MemoryID); err != nil {
		return err
	}
	storageReq := storage.DeleteMemoryFromVaultRequest(req)
	return s.storage.DeleteMemoryFromVault(ctx, storageReq)
}

// Validation helpers
func (s *Service) validateCreateVaultRequest(req CreateVaultRequest) error {
	if req.UserID == "" {
		return memcore.NewValidationError("userID", "user ID is required")
	}
	if req.Title == "" {
		return memcore.NewValidationError("title", "title is required")
	}
	if len(req.Title) > 50 {
		return memcore.NewValidationError("title", "title exceeds 50 characters")
	}
	if !titleRx.MatchString(req.Title) {
		return memcore.NewValidationError("title", "title contains invalid characters; allowed letters, digits, hyphen")
	}
	if req.Description != nil && len(*req.Description) > 500 {
		return memcore.NewValidationError("description", "description exceeds 500 characters")
	}
	return nil
}

func (s *Service) validateMemoryAssociationRequest(userID string, vaultID uuid.UUID, memoryID string) error {
	if userID == "" {
		return memcore.NewValidationError("userID", "user ID is required")
	}
	if vaultID == uuid.Nil {
		return memcore.NewValidationError("vaultID", "vault ID is required")
	}
	if memoryID == "" {
		return memcore.NewValidationError("memoryID", "memory ID is required")
	}
	return nil
}

// title validation regex shared by vault & memory services (ASCII letters, digits, space, hyphen, underscore, apostrophe)
var titleRx = regexp.MustCompile(`^[A-Za-z0-9\-]+$`)

// allowed lowercase letters, digits, hyphen, 1-50 chars
