package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dariamoshkina/gopherMart/internal/handler/mocks"
	"github.com/dariamoshkina/gopherMart/internal/middleware"
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
)

func withUser(r *http.Request, userID int64) *http.Request {
	return r.WithContext(middleware.ContextWithUserID(r.Context(), userID))
}

func TestOrdersHandler_Submit_Accepted(t *testing.T) {
	svc := mocks.NewMockOrdersService(t)
	h := NewOrdersHandler(svc)
	svc.On("SubmitOrder", mock.Anything, int64(1), "12345678903").Return(nil)

	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345678903")), 1)
	rec := httptest.NewRecorder()
	h.Submit(rec, r)
	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestOrdersHandler_Submit_SameUser(t *testing.T) {
	svc := mocks.NewMockOrdersService(t)
	h := NewOrdersHandler(svc)
	svc.On("SubmitOrder", mock.Anything, int64(1), "12345678903").Return(service.ErrOrderOwnedBySameUser)

	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345678903")), 1)
	rec := httptest.NewRecorder()
	h.Submit(rec, r)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestOrdersHandler_Submit_OtherUser(t *testing.T) {
	svc := mocks.NewMockOrdersService(t)
	h := NewOrdersHandler(svc)
	svc.On("SubmitOrder", mock.Anything, int64(1), "12345678903").Return(service.ErrOrderOwnedByOtherUser)

	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345678903")), 1)
	rec := httptest.NewRecorder()
	h.Submit(rec, r)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestOrdersHandler_Submit_LuhnFailure(t *testing.T) {
	svc := mocks.NewMockOrdersService(t)
	h := NewOrdersHandler(svc)
	svc.On("SubmitOrder", mock.Anything, int64(1), "12345678903").Return(service.ErrInvalidOrderNumber)

	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345678903")), 1)
	rec := httptest.NewRecorder()
	h.Submit(rec, r)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestOrdersHandler_Submit_EmptyBody(t *testing.T) {
	h := NewOrdersHandler(mocks.NewMockOrdersService(t))
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("")), 1)
	rec := httptest.NewRecorder()
	h.Submit(rec, r)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOrdersHandler_Submit_NonDigit(t *testing.T) {
	h := NewOrdersHandler(mocks.NewMockOrdersService(t))
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("abc123")), 1)
	rec := httptest.NewRecorder()
	h.Submit(rec, r)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOrdersHandler_Submit_NoAuth(t *testing.T) {
	h := NewOrdersHandler(mocks.NewMockOrdersService(t))
	rec := httptest.NewRecorder()
	h.Submit(rec, httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("12345678903")))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOrdersHandler_List_OK(t *testing.T) {
	svc := mocks.NewMockOrdersService(t)
	h := NewOrdersHandler(svc)

	orders := []*model.Order{
		{OrderNumber: "12345678903", Status: model.OrderStatusProcessed, Accrual: new(int64(10000)), UploadedAt: time.Now()},
	}
	svc.On("ListOrders", mock.Anything, int64(1)).Return(orders, nil)

	r := withUser(httptest.NewRequest(http.MethodGet, "/api/user/orders", nil), 1)
	rec := httptest.NewRecorder()
	h.List(rec, r)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp []orderResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "12345678903", resp[0].Number)
	require.NotNil(t, resp[0].Accrual)
	assert.Equal(t, 100.0, *resp[0].Accrual)
}

func TestOrdersHandler_List_NoAccrualOnNewOrder(t *testing.T) {
	svc := mocks.NewMockOrdersService(t)
	h := NewOrdersHandler(svc)

	orders := []*model.Order{
		{OrderNumber: "12345678903", Status: model.OrderStatusNew, Accrual: new(int64(0)), UploadedAt: time.Now()},
	}
	svc.On("ListOrders", mock.Anything, int64(1)).Return(orders, nil)

	r := withUser(httptest.NewRequest(http.MethodGet, "/api/user/orders", nil), 1)
	rec := httptest.NewRecorder()
	h.List(rec, r)

	var resp []orderResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Nil(t, resp[0].Accrual)
}

func TestOrdersHandler_List_Empty(t *testing.T) {
	svc := mocks.NewMockOrdersService(t)
	h := NewOrdersHandler(svc)
	svc.On("ListOrders", mock.Anything, int64(1)).Return([]*model.Order{}, nil)

	r := withUser(httptest.NewRequest(http.MethodGet, "/api/user/orders", nil), 1)
	rec := httptest.NewRecorder()
	h.List(rec, r)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestOrdersHandler_List_NoAuth(t *testing.T) {
	h := NewOrdersHandler(mocks.NewMockOrdersService(t))
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/api/user/orders", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
