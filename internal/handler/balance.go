package handler

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/dariamoshkina/gopherMart/internal/middleware"
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
)

type BalanceService interface {
	GetBalance(ctx context.Context, userID int64) (*model.Balance, error)
	Withdraw(ctx context.Context, userID int64, orderNumber string, sum int64) error
	ListWithdrawals(ctx context.Context, userID int64) ([]*model.Withdrawal, error)
}

type BalanceHandler struct {
	svc BalanceService
}

func NewBalanceHandler(svc BalanceService) *BalanceHandler {
	return &BalanceHandler{svc: svc}
}

type balanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type withdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	balance, err := h.svc.GetBalance(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balanceResponse{
		Current:   float64(balance.Current) / 100,
		Withdrawn: float64(balance.Withdrawn) / 100,
	})
}

func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Order == "" || req.Sum <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sum := int64(math.Round(req.Sum * 100))
	err := h.svc.Withdraw(r.Context(), userID, req.Order, sum)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, service.ErrInvalidOrderNumber):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Is(err, service.ErrInsufficientBalance):
		w.WriteHeader(http.StatusPaymentRequired)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *BalanceHandler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.svc.ListWithdrawals(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]withdrawalResponse, 0, len(withdrawals))
	for _, withdrawal := range withdrawals {
		resp = append(resp, withdrawalResponse{
			Order:       withdrawal.OrderNumber,
			Sum:         float64(withdrawal.Sum) / 100,
			ProcessedAt: withdrawal.ProcessedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
