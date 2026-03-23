package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/dariamoshkina/gopherMart/internal/client/accrual"
	"github.com/dariamoshkina/gopherMart/internal/model"
)

type mockOrderRepo struct{ mock.Mock }

func (m *mockOrderRepo) GetPending(ctx context.Context, limit int) ([]*model.Order, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Order), args.Error(1)
}
func (m *mockOrderRepo) UpdateStatus(ctx context.Context, orderID int64, status string, a *int64) error {
	return m.Called(ctx, orderID, status, a).Error(0)
}
func (m *mockOrderRepo) MarkProcessedWithCredit(ctx context.Context, orderID, userID int64, a *int64) error {
	return m.Called(ctx, orderID, userID, a).Error(0)
}

type mockAccrualClient struct{ mock.Mock }

func (m *mockAccrualClient) GetOrder(ctx context.Context, orderNumber string) (*accrual.AccrualResult, error) {
	args := m.Called(ctx, orderNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*accrual.AccrualResult), args.Error(1)
}

func newTestPoller(orders *mockOrderRepo, client *mockAccrualClient) *Poller {
	return New(orders, client, 100*time.Millisecond, zap.NewNop())
}

func TestPoller_ProcessOrder_NotRegistered(t *testing.T) {
	orders := &mockOrderRepo{}
	client := &mockAccrualClient{}
	p := newTestPoller(orders, client)

	order := &model.Order{ID: 1, OrderNumber: "12345678903"}
	client.On("GetOrder", mock.Anything, "12345678903").Return(nil, accrual.ErrNotRegistered)

	err := p.processOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	orders.AssertNotCalled(t, "UpdateStatus")
}

func TestPoller_ProcessOrder_Processing(t *testing.T) {
	orders := &mockOrderRepo{}
	client := &mockAccrualClient{}
	p := newTestPoller(orders, client)

	order := &model.Order{ID: 1, OrderNumber: "12345678903"}
	client.On("GetOrder", mock.Anything, "12345678903").
		Return(&accrual.AccrualResult{Status: "PROCESSING"}, nil)
	orders.On("UpdateStatus", mock.Anything, int64(1), model.OrderStatusProcessing, (*int64)(nil)).
		Return(nil)

	err := p.processOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	orders.AssertExpectations(t)
}

func TestPoller_ProcessOrder_Invalid(t *testing.T) {
	orders := &mockOrderRepo{}
	client := &mockAccrualClient{}
	p := newTestPoller(orders, client)

	order := &model.Order{ID: 1, OrderNumber: "12345678903"}
	client.On("GetOrder", mock.Anything, "12345678903").
		Return(&accrual.AccrualResult{Status: "INVALID"}, nil)
	orders.On("UpdateStatus", mock.Anything, int64(1), model.OrderStatusInvalid, (*int64)(nil)).
		Return(nil)

	err := p.processOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	orders.AssertExpectations(t)
}

func TestPoller_ProcessOrder_Processed(t *testing.T) {
	orders := &mockOrderRepo{}
	client := &mockAccrualClient{}
	p := newTestPoller(orders, client)

	clientAccrual := 200.0
	repoAccrual := int64(20000)
	order := &model.Order{ID: 1, UserID: 42, OrderNumber: "12345678903"}
	client.On("GetOrder", mock.Anything, "12345678903").
		Return(&accrual.AccrualResult{Status: "PROCESSED", Accrual: &clientAccrual}, nil)
	orders.On("MarkProcessedWithCredit", mock.Anything, int64(1), int64(42), &repoAccrual).Return(nil)

	err := p.processOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	orders.AssertExpectations(t)
}

func TestPoller_Poll_RepoError(t *testing.T) {
	orders := &mockOrderRepo{}
	client := &mockAccrualClient{}
	p := newTestPoller(orders, client)

	orders.On("GetPending", mock.Anything, 100).Return(nil, errors.New("db down"))

	p.poll(context.Background())
	orders.AssertExpectations(t)
}
