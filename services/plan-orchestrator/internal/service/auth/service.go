package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	authutil "github.com/travel-agent/services/plan-orchestrator/internal/auth"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	userrepo "github.com/travel-agent/services/plan-orchestrator/internal/repository/user"
	"github.com/travel-agent/shared/contracts"
)

type Service struct {
	repository   userrepo.Repository
	tokenManager *authutil.TokenManager
}

func NewService(repository userrepo.Repository, tokenManager *authutil.TokenManager) *Service {
	return &Service{
		repository:   repository,
		tokenManager: tokenManager,
	}
}

func (s *Service) Register(ctx context.Context, req contracts.RegisterRequest) (contracts.AuthResponse, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return contracts.AuthResponse{}, fmt.Errorf("username and password are required")
	}

	hashedPassword, err := authutil.HashPassword(password)
	if err != nil {
		return contracts.AuthResponse{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repository.Create(ctx, domain.User{
		Username:     username,
		PasswordHash: hashedPassword,
		Nickname:     strings.TrimSpace(req.Nickname),
		Status:       1,
	})
	if err != nil {
		return contracts.AuthResponse{}, err
	}

	return s.buildAuthResponse(user)
}

func (s *Service) Login(ctx context.Context, req contracts.LoginRequest) (contracts.AuthResponse, error) {
	user, err := s.repository.GetByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return contracts.AuthResponse{}, fmt.Errorf("invalid username or password")
		}
		return contracts.AuthResponse{}, err
	}

	if user.Status != 1 {
		return contracts.AuthResponse{}, fmt.Errorf("user is disabled")
	}
	if err := authutil.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return contracts.AuthResponse{}, fmt.Errorf("invalid username or password")
	}

	return s.buildAuthResponse(user)
}

func (s *Service) GetCurrentUser(ctx context.Context, userID int64) (contracts.UserProfile, error) {
	user, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return contracts.UserProfile{}, err
	}
	return toUserProfile(user), nil
}

func (s *Service) buildAuthResponse(user domain.User) (contracts.AuthResponse, error) {
	token, expiresIn, err := s.tokenManager.Generate(user.ID, user.Username)
	if err != nil {
		return contracts.AuthResponse{}, fmt.Errorf("generate token: %w", err)
	}

	return contracts.AuthResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User:        toUserProfile(user),
	}, nil
}

func toUserProfile(user domain.User) contracts.UserProfile {
	return contracts.UserProfile{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
	}
}
