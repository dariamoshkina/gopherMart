package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service/mocks"
)

func TestBalanceService_GetBalance(t *testing.T) {
	repo := mocks.NewMockBalanceRepository(t)
	svc := NewBalanceService(repo)

	b := &model.Balance{UserID: 1, Current: 10000, Withdrawn: 2000}
	repo.On("GetOrCreate", mock.Anything, int64(1)).Return(b, nil).Once()

	result, err := svc.GetBalance(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, b, result)
	repo.AssertExpectations(t)
}

func TestBalanceService_Withdraw_Success(t *testing.T) {
	repo := mocks.NewMockBalanceRepository(t)
	svc := NewBalanceService(repo)

	repo.On("Withdraw", mock.Anything, int64(1), "2377225624", int64(5000)).Return(nil).Once()

	err := svc.Withdraw(context.Background(), 1, "2377225624", 5000)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBalanceService_Withdraw_LuhnFailure(t *testing.T) {
	repo := mocks.NewMockBalanceRepository(t)
	svc := NewBalanceService(repo)

	err := svc.Withdraw(context.Background(), 1, "1234567890", 5000)
	assert.ErrorIs(t, err, ErrInvalidOrderNumber)
	repo.AssertNotCalled(t, "Withdraw")
}

func TestBalanceService_Withdraw_InsufficientBalance(t *testing.T) {
	repo := mocks.NewMockBalanceRepository(t)
	svc := NewBalanceService(repo)

	repo.On("Withdraw", mock.Anything, int64(1), "2377225624", int64(50000)).
		Return(ErrInsufficientBalance).Once()

	err := svc.Withdraw(context.Background(), 1, "2377225624", 50000)
	assert.ErrorIs(t, err, ErrInsufficientBalance)
	repo.AssertExpectations(t)
}

func TestBalanceService_ListWithdrawals(t *testing.T) {
	repo := mocks.NewMockBalanceRepository(t)
	svc := NewBalanceService(repo)

	ws := []*model.Withdrawal{
		{ID: 1, UserID: 1, OrderNumber: "2377225624", Sum: 5000, ProcessedAt: time.Now()},
	}
	repo.On("ListWithdrawalsByUserID", mock.Anything, int64(1)).Return(ws, nil).Once()

	result, err := svc.ListWithdrawals(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

func TestBalanceService_ListWithdrawals_Empty(t *testing.T) {
	repo := mocks.NewMockBalanceRepository(t)
	svc := NewBalanceService(repo)

	repo.On("ListWithdrawalsByUserID", mock.Anything, int64(1)).
		Return([]*model.Withdrawal{}, nil).Once()

	result, err := svc.ListWithdrawals(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, result)
}
