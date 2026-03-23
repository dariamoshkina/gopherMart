package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrder_200(t *testing.T) {
	accrual := 150.5
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/orders/12345678903", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AccrualResult{
			Order:   "12345678903",
			Status:  "PROCESSED",
			Accrual: &accrual,
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	result, err := c.GetOrder(context.Background(), "12345678903")
	require.NoError(t, err)
	assert.Equal(t, "PROCESSED", result.Status)
	require.NotNil(t, result.Accrual)
	assert.Equal(t, 150.5, *result.Accrual)
}

func TestGetOrder_200_NoAccrual(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AccrualResult{Order: "12345678903", Status: "PROCESSING"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	result, err := c.GetOrder(context.Background(), "12345678903")
	require.NoError(t, err)
	assert.Equal(t, "PROCESSING", result.Status)
	assert.Nil(t, result.Accrual)
}

func TestGetOrder_204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.GetOrder(context.Background(), "12345678903")
	assert.ErrorIs(t, err, ErrNotRegistered)
}

func TestGetOrder_429_WithRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.GetOrder(context.Background(), "12345678903")
	require.Error(t, err)

	var rl *RateLimitError
	require.True(t, errors.As(err, &rl))
	assert.Equal(t, 30, int(rl.RetryAfter.Seconds()))
}

func TestGetOrder_429_MissingRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.GetOrder(context.Background(), "12345678903")
	var rl *RateLimitError
	require.True(t, errors.As(err, &rl))
	assert.Equal(t, 60, int(rl.RetryAfter.Seconds()))
}

func TestGetOrder_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.GetOrder(context.Background(), "12345678903")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrNotRegistered))
}
