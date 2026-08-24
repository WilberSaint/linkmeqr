package services

import (
	"context"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/utils"
)

type ClientService struct {
	users *repository.UserRepository
}

func NewClientService(users *repository.UserRepository) *ClientService {
	return &ClientService{users: users}
}

func (s *ClientService) Create(ctx context.Context, email, password, fullName string, phone *string) (*models.User, error) {
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		Role:         models.RoleClient,
		FullName:     fullName,
		Phone:        phone,
		IsActive:     true,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *ClientService) List(ctx context.Context) ([]models.User, error) {
	return s.users.List(ctx, models.RoleClient)
}

func (s *ClientService) Get(ctx context.Context, id string) (*models.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *ClientService) SetActive(ctx context.Context, id string, active bool) error {
	return s.users.SetActive(ctx, id, active)
}

func (s *ClientService) Update(ctx context.Context, user *models.User) error {
	return s.users.Update(ctx, user)
}
