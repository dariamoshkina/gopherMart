package service

//go:generate mockery --name=BalanceRepository --output=./mocks --outpkg=mocks --filename=mock_balance_repository.go --with-expecter=false

import (
	"context"
	"errors"
	"fmt"

	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/pkg/luhn"
)

type BalanceRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*model.Balance, error)
	GetOrCreate(ctx context.Context, userID int64) (*model.Balance, error)
	Credit(ctx context.Context, userID int64, amount int64) error
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum int64) error
	ListWithdrawalsByUserID(ctx context.Context, userID int64) ([]*model.Withdrawal, error)
}

type BalanceService struct {
	balanceRepo BalanceRepository
}

func NewBalanceService(balanceRepo BalanceRepository) *BalanceService {
	return &BalanceService{balanceRepo: balanceRepo}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (*model.Balance, error) {
	balance, err := s.balanceRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

func (s *BalanceService) Withdraw(ctx context.Context, userID int64, orderNumber string, sum int64) error {
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}
	err := s.balanceRepo.Withdraw(ctx, userID, orderNumber, sum)
	if err != nil {
		if errors.Is(err, ErrInsufficientBalance) {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("withdraw: %w", err)
	}
	return nil
}

func (s *BalanceService) ListWithdrawals(ctx context.Context, userID int64) ([]*model.Withdrawal, error) {
	withdrawals, err := s.balanceRepo.ListWithdrawalsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals: %w", err)
	}
	return withdrawals, nil
}
