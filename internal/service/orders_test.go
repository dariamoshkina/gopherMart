package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service/mocks"
)

func TestOrdersService_SubmitOrder_Success(t *testing.T) {
	repo := mocks.NewMockOrderRepository(t)
	svc := NewOrdersService(repo)

	order := &model.Order{ID: 1, UserID: 42, OrderNumber: "12345678903"}
	repo.On("Create", mock.Anything, int64(42), "12345678903").Return(order, nil).Once()

	err := svc.SubmitOrder(context.Background(), 42, "12345678903")
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestOrdersService_SubmitOrder_LuhnFailure(t *testing.T) {
	repo := mocks.NewMockOrderRepository(t)
	svc := NewOrdersService(repo)

	err := svc.SubmitOrder(context.Background(), 42, "12345678900")
	assert.ErrorIs(t, err, ErrInvalidOrderNumber)
	repo.AssertNotCalled(t, "Create")
}

func TestOrdersService_SubmitOrder_SameUser(t *testing.T) {
	repo := mocks.NewMockOrderRepository(t)
	svc := NewOrdersService(repo)

	repo.On("Create", mock.Anything, int64(42), "12345678903").Return(nil, ErrDuplicateOrderNumber).Once()
	repo.On("GetByOrderNumber", mock.Anything, "12345678903").
		Return(&model.Order{UserID: 42}, nil).Once()

	err := svc.SubmitOrder(context.Background(), 42, "12345678903")
	assert.ErrorIs(t, err, ErrOrderOwnedBySameUser)
	repo.AssertExpectations(t)
}

func TestOrdersService_SubmitOrder_OtherUser(t *testing.T) {
	repo := mocks.NewMockOrderRepository(t)
	svc := NewOrdersService(repo)

	repo.On("Create", mock.Anything, int64(42), "12345678903").Return(nil, ErrDuplicateOrderNumber).Once()
	repo.On("GetByOrderNumber", mock.Anything, "12345678903").
		Return(&model.Order{UserID: 99}, nil).Once()

	err := svc.SubmitOrder(context.Background(), 42, "12345678903")
	assert.ErrorIs(t, err, ErrOrderOwnedByOtherUser)
	repo.AssertExpectations(t)
}

func TestOrdersService_ListOrders(t *testing.T) {
	repo := mocks.NewMockOrderRepository(t)
	svc := NewOrdersService(repo)

	orders := []*model.Order{
		{ID: 1, UserID: 42, OrderNumber: "12345678903", Status: model.OrderStatusNew, UploadedAt: time.Now()},
	}
	repo.On("GetByUserID", mock.Anything, int64(42)).Return(orders, nil).Once()

	result, err := svc.ListOrders(context.Background(), 42)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

func TestOrdersService_ListOrders_Empty(t *testing.T) {
	repo := mocks.NewMockOrderRepository(t)
	svc := NewOrdersService(repo)

	repo.On("GetByUserID", mock.Anything, int64(42)).Return([]*model.Order{}, nil).Once()

	result, err := svc.ListOrders(context.Background(), 42)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestOrdersService_SubmitOrder_RepoError(t *testing.T) {
	repo := mocks.NewMockOrderRepository(t)
	svc := NewOrdersService(repo)

	repo.On("Create", mock.Anything, int64(42), "12345678903").
		Return(nil, errors.New("db down")).Once()

	err := svc.SubmitOrder(context.Background(), 42, "12345678903")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrInvalidOrderNumber))
}
