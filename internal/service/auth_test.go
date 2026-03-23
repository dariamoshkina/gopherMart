package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"golang.org/x/crypto/bcrypt"

	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service/mocks"
)

func TestAuthService_Register_Success(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockUserRepository(t)
	svc := NewAuthService(repo)

	login, password := "user1", "secret"
	created := &model.User{ID: 1, Login: login, PasswordHash: "hash", CreatedAt: time.Now()}

	repo.On("Create", ctx, login, mock.AnythingOfType("string")).Return(created, nil).Once()

	user, err := svc.Register(ctx, login, password)
	require.NoError(t, err)
	assert.Equal(t, created.ID, user.ID)
	assert.Equal(t, created.Login, user.Login)
	repo.AssertExpectations(t)
}

func TestAuthService_Register_LoginTaken(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockUserRepository(t)
	svc := NewAuthService(repo)

	login, password := "taken", "secret"

	repo.On("Create", ctx, login, mock.AnythingOfType("string")).Return(nil, ErrLoginTaken).Once()

	user, err := svc.Register(ctx, login, password)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoginTaken))
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}

func TestAuthService_Login_Success(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockUserRepository(t)
	svc := NewAuthService(repo)

	login, password := "user1", "secret"
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	stored := &model.User{ID: 1, Login: login, PasswordHash: string(hashed), CreatedAt: time.Now()}

	repo.On("GetByLogin", ctx, login).Return(stored, nil).Once()

	user, err := svc.Login(ctx, login, password)
	require.NoError(t, err)
	assert.Equal(t, stored.ID, user.ID)
	assert.Equal(t, stored.Login, user.Login)
	repo.AssertExpectations(t)
}

func TestAuthService_Login_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockUserRepository(t)
	svc := NewAuthService(repo)

	repo.On("GetByLogin", ctx, "nobody").Return(nil, ErrUserNotFound).Once()

	user, err := svc.Login(ctx, "nobody", "pass")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockUserRepository(t)
	svc := NewAuthService(repo)

	login := "user1"
	stored := &model.User{ID: 1, Login: login, PasswordHash: "wrong-hash", CreatedAt: time.Now()}

	repo.On("GetByLogin", ctx, login).Return(stored, nil).Once()

	user, err := svc.Login(ctx, login, "correct-password")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
	assert.Nil(t, user)
	repo.AssertExpectations(t)
}
