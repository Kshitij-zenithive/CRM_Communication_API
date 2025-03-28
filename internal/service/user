// internal/service/user.go (Added missing UserService)

package service

import (
	"context"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"fmt"

	"github.com/google/uuid"
)

type UserService struct {
	repo *repository.Repository
}

func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.repo.GetUserByEmail(ctx, email)
}

// UpdateUser updates an existing user in the repository.
func (s *UserService) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {

	existingUser, err := s.repo.GetUserByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Update fields as necessary.  Be careful about what you allow to be updated.
	existingUser.Name = user.Name
	existingUser.Email = user.Email // Consider if email should be updatable.
	existingUser.Avatar = user.Avatar
	existingUser.Role = user.Role //Be careful before updating

	if user.Password != "" {
		//Update password
		err = existingUser.SetPassword(user.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to set password: %w", err)
		}
	}

	if err := s.repo.UpdateUser(ctx, existingUser); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return existingUser, nil
}

// DeleteUser is not implemented
func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return fmt.Errorf("DeleteUser is not implemented. Consider soft deletes or account deactivation")
}
