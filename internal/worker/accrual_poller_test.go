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
	"github.com/dariamoshkina/gopherMart/internal/worker/mocks"
)

func newTestPoller(orders *mocks.MockOrderRepo, client *mocks.MockAccrualClient) *Poller {
	return New(orders, client, 100*time.Millisecond, 1, zap.NewNop())
}

func TestPoller_ProcessOrder_NotRegistered(t *testing.T) {
	orders := mocks.NewMockOrderRepo(t)
	client := mocks.NewMockAccrualClient(t)
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
	orders := mocks.NewMockOrderRepo(t)
	client := mocks.NewMockAccrualClient(t)
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
	orders := mocks.NewMockOrderRepo(t)
	client := mocks.NewMockAccrualClient(t)
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
	orders := mocks.NewMockOrderRepo(t)
	client := mocks.NewMockAccrualClient(t)
	p := newTestPoller(orders, client)

	order := &model.Order{ID: 1, UserID: 42, OrderNumber: "12345678903"}
	client.On("GetOrder", mock.Anything, "12345678903").
		Return(&accrual.AccrualResult{Status: "PROCESSED", Accrual: new(200.0)}, nil)
	orders.On("MarkProcessedWithCredit", mock.Anything, int64(1), int64(42), new(int64(20000))).Return(nil)

	err := p.processOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	orders.AssertExpectations(t)
}

func TestPoller_Poll_RepoError(t *testing.T) {
	orders := mocks.NewMockOrderRepo(t)
	client := mocks.NewMockAccrualClient(t)
	p := newTestPoller(orders, client)

	orders.On("GetPending", mock.Anything, 100).Return(nil, errors.New("db down"))

	jobs := make(chan *model.Order, 1)
	p.poll(context.Background(), jobs)
	orders.AssertExpectations(t)
}

func TestPoller_PauseFor_Waits(t *testing.T) {
	p := New(nil, nil, time.Second, 1, zap.NewNop())
	p.pauseFor(50 * time.Millisecond)

	start := time.Now()
	p.waitIfPaused(context.Background())
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("waitIfPaused returned too early: %v", elapsed)
	}
}

func TestPoller_Run_ShutdownWhilePaused(t *testing.T) {
	orders := mocks.NewMockOrderRepo(t)
	client := mocks.NewMockAccrualClient(t)
	p := New(orders, client, 10*time.Millisecond, 1, zap.NewNop())

	order := &model.Order{ID: 1, OrderNumber: "12345678903"}
	orders.On("GetPending", mock.Anything, 100).Return([]*model.Order{order}, nil).Maybe()
	client.On("GetOrder", mock.Anything, "12345678903").
		Return(nil, &accrual.RateLimitError{RetryAfter: 10 * time.Second}).Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.Run(ctx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not shut down promptly while paused")
	}
}
