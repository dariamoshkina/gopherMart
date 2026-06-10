package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/dariamoshkina/gopherMart/internal/infra"
)

var ErrNotRegistered = errors.New("order not registered in accrual system")

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %s", e.RetryAfter)
}

type AccrualResult struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func New(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		logger:  logger,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: infra.NewRetryTransport(http.DefaultTransport, 3, 200*time.Millisecond, 5*time.Second),
		},
	}
}

func (c *Client) GetOrder(ctx context.Context, orderNumber string) (*AccrualResult, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build accrual request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("accrual request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var result AccrualResult
		if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode accrual response: %w", err)
		}
		return &result, nil
	case http.StatusNoContent:
		return nil, ErrNotRegistered
	case http.StatusTooManyRequests:
		retryAfter, err := parseRetryAfter(resp.Header.Get("Retry-After"))
		if err != nil {
			c.logger.Warn("parse Retry-After header", zap.Error(err))
		}
		return nil, &RateLimitError{RetryAfter: retryAfter}
	default:
		return nil, fmt.Errorf("accrual service returned %d", resp.StatusCode)
	}
}

func parseRetryAfter(headerVal string) (time.Duration, error) {
	if s, err := strconv.Atoi(headerVal); err == nil && s > 0 {
		return time.Duration(s) * time.Second, nil
	}
	return 60 * time.Second, fmt.Errorf("invalid Retry-After header %q, defaulting to 60s", headerVal)
}
