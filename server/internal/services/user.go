package services

import (
	"context"

	"github.com/mycelian/mycelian-memory/server/internal/model"
	"github.com/mycelian/mycelian-memory/server/internal/store"
)

// UserService handles user-related operations.
type UserService struct {
	store store.Store
}

func NewUserService(s store.Store) *UserService { return &UserService{store: s} }

func (s *UserService) CreateUser(ctx context.Context, u *model.User) (*model.User, error) {
	return s.store.Users().Create(ctx, u)
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	return s.store.Users().Get(ctx, userID)
}
