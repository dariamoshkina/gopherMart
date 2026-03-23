package service

//go:generate mockery --name=UserRepository --output=./mocks --outpkg=mocks --filename=mock_user_repository.go --with-expecter=false

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/dariamoshkina/gopherMart/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}

type AuthService struct {
	userRepo UserRepository
}

func NewAuthService(userRepo UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Register(ctx context.Context, login, password string) (*model.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	passwordHash := string(hashed)
	user, err := s.userRepo.Create(ctx, login, passwordHash)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, login, password string) (*model.User, error) {
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}
