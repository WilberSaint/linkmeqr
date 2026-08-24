package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/utils"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountInactive    = errors.New("account inactive")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
)

type AuthService struct {
	users         *repository.UserRepository
	refreshTokens *repository.RefreshTokenRepository
	jwtSecret     string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewAuthService(users *repository.UserRepository, refreshTokens *repository.RefreshTokenRepository, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		jwtSecret:     jwtSecret,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	User         *models.User
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	valid, err := utils.VerifyPassword(password, user.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	return s.issueTokenPair(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	hash := utils.HashToken(refreshToken)
	stored, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}

	if !s.refreshTokens.IsValid(stored) {
		return nil, ErrInvalidRefresh
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, ErrInvalidRefresh
	}
	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	// Rotate: revoke the used refresh token before issuing a new pair.
	if err := s.refreshTokens.Revoke(ctx, hash); err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := utils.HashToken(refreshToken)
	return s.refreshTokens.Revoke(ctx, hash)
}

func (s *AuthService) issueTokenPair(ctx context.Context, user *models.User) (*TokenPair, error) {
	accessToken, err := utils.GenerateAccessToken(s.jwtSecret, user.ID, string(user.Role), s.accessTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	record := &models.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: utils.HashToken(refreshToken),
		ExpiresAt: time.Now().UTC().Add(s.refreshTTL),
	}
	if err := s.refreshTokens.Create(ctx, record); err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, User: user}, nil
}
