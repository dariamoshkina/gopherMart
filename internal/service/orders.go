package service

//go:generate mockery --name=OrderRepository --output=./mocks --outpkg=mocks --filename=mock_order_repository.go --with-expecter=false

import (
	"context"
	"errors"
	"fmt"

	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/pkg/luhn"
)

type OrderRepository interface {
	Create(ctx context.Context, userID int64, orderNumber string) (*model.Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]*model.Order, error)
	GetByOrderNumber(ctx context.Context, orderNumber string) (*model.Order, error)
	GetPending(ctx context.Context, limit int) ([]*model.Order, error)
	UpdateStatus(ctx context.Context, orderID int64, status string, accrual *int64) error
}

type OrdersService struct {
	orderRepo OrderRepository
}

func NewOrdersService(orderRepo OrderRepository) *OrdersService {
	return &OrdersService{orderRepo: orderRepo}
}

func (s *OrdersService) SubmitOrder(ctx context.Context, userID int64, orderNumber string) error {
	if !luhn.Validate(orderNumber) {
		return ErrInvalidOrderNumber
	}
	_, err := s.orderRepo.Create(ctx, userID, orderNumber)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrDuplicateOrderNumber) {
		return fmt.Errorf("create order: %w", err)
	}
	existing, err := s.orderRepo.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		return fmt.Errorf("fetch existing order: %w", err)
	}
	if existing.UserID == userID {
		return ErrOrderOwnedBySameUser
	}
	return ErrOrderOwnedByOtherUser
}

func (s *OrdersService) ListOrders(ctx context.Context, userID int64) ([]*model.Order, error) {
	orders, err := s.orderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
}
