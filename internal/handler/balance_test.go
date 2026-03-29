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
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
)

func TestBalanceHandler_GetBalance_OK(t *testing.T) {
	svc := mocks.NewMockBalanceService(t)
	h := NewBalanceHandler(svc)

	svc.On("GetBalance", mock.Anything, int64(1)).Return(&model.Balance{Current: 15000, Withdrawn: 3000}, nil)

	r := withUser(httptest.NewRequest(http.MethodGet, "/api/user/balance", nil), 1)
	rec := httptest.NewRecorder()
	h.GetBalance(rec, r)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp balanceResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, 150.0, resp.Current)
	assert.Equal(t, 30.0, resp.Withdrawn)
}

func TestBalanceHandler_GetBalance_NoAuth(t *testing.T) {
	h := NewBalanceHandler(mocks.NewMockBalanceService(t))
	rec := httptest.NewRecorder()
	h.GetBalance(rec, httptest.NewRequest(http.MethodGet, "/api/user/balance", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBalanceHandler_Withdraw_OK(t *testing.T) {
	svc := mocks.NewMockBalanceService(t)
	h := NewBalanceHandler(svc)

	svc.On("Withdraw", mock.Anything, int64(1), "2377225624", int64(5000)).Return(nil)

	body := `{"order":"2377225624","sum":50}`
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(body)), 1)
	rec := httptest.NewRecorder()
	h.Withdraw(rec, r)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBalanceHandler_Withdraw_InsufficientBalance(t *testing.T) {
	svc := mocks.NewMockBalanceService(t)
	h := NewBalanceHandler(svc)

	svc.On("Withdraw", mock.Anything, int64(1), "2377225624", int64(99900)).Return(service.ErrInsufficientBalance)

	body := `{"order":"2377225624","sum":999}`
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(body)), 1)
	rec := httptest.NewRecorder()
	h.Withdraw(rec, r)
	assert.Equal(t, http.StatusPaymentRequired, rec.Code)
}

func TestBalanceHandler_Withdraw_LuhnFailure(t *testing.T) {
	svc := mocks.NewMockBalanceService(t)
	h := NewBalanceHandler(svc)

	svc.On("Withdraw", mock.Anything, int64(1), "1234567890", int64(1000)).Return(service.ErrInvalidOrderNumber)

	body := `{"order":"1234567890","sum":10}`
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(body)), 1)
	rec := httptest.NewRecorder()
	h.Withdraw(rec, r)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestBalanceHandler_Withdraw_BadRequest(t *testing.T) {
	h := NewBalanceHandler(mocks.NewMockBalanceService(t))
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{invalid`)), 1)
	rec := httptest.NewRecorder()
	h.Withdraw(rec, r)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBalanceHandler_Withdraw_NoAuth(t *testing.T) {
	h := NewBalanceHandler(mocks.NewMockBalanceService(t))
	rec := httptest.NewRecorder()
	h.Withdraw(rec, httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(`{"order":"2377225624","sum":10}`)))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBalanceHandler_ListWithdrawals_OK(t *testing.T) {
	svc := mocks.NewMockBalanceService(t)
	h := NewBalanceHandler(svc)

	ws := []*model.Withdrawal{{OrderNumber: "2377225624", Sum: 5000, ProcessedAt: time.Now()}}
	svc.On("ListWithdrawals", mock.Anything, int64(1)).Return(ws, nil)

	r := withUser(httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil), 1)
	rec := httptest.NewRecorder()
	h.ListWithdrawals(rec, r)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp []withdrawalResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "2377225624", resp[0].Order)
	assert.Equal(t, 50.0, resp[0].Sum)
}

func TestBalanceHandler_ListWithdrawals_Empty(t *testing.T) {
	svc := mocks.NewMockBalanceService(t)
	h := NewBalanceHandler(svc)
	svc.On("ListWithdrawals", mock.Anything, int64(1)).Return([]*model.Withdrawal{}, nil)

	r := withUser(httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil), 1)
	rec := httptest.NewRecorder()
	h.ListWithdrawals(rec, r)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestBalanceHandler_ListWithdrawals_NoAuth(t *testing.T) {
	h := NewBalanceHandler(mocks.NewMockBalanceService(t))
	rec := httptest.NewRecorder()
	h.ListWithdrawals(rec, httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
