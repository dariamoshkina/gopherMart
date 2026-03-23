package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/dariamoshkina/gopherMart/internal/middleware"
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
)

type OrdersService interface {
	SubmitOrder(ctx context.Context, userID int64, orderNumber string) error
	ListOrders(ctx context.Context, userID int64) ([]*model.Order, error)
}

type OrdersHandler struct {
	svc OrdersService
}

func NewOrdersHandler(svc OrdersService) *OrdersHandler {
	return &OrdersHandler{svc: svc}
}

type orderResponse struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`
}

func (h *OrdersHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	orderNumber := strings.TrimSpace(string(body))
	if orderNumber == "" || !isAllDigits(orderNumber) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.svc.SubmitOrder(r.Context(), userID, orderNumber)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, service.ErrInvalidOrderNumber):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Is(err, service.ErrOrderOwnedBySameUser):
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, service.ErrOrderOwnedByOtherUser):
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *OrdersHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	orders, err := h.svc.ListOrders(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		item := orderResponse{
			Number:     order.OrderNumber,
			Status:     order.Status,
			UploadedAt: order.UploadedAt.Format(time.RFC3339),
		}

		if order.Accrual != nil && *order.Accrual > 0 {
			accrual := float64(*order.Accrual) / 100
			item.Accrual = &accrual
		}
		resp = append(resp, item)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
