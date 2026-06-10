package infra

import (
	"context"
	"io"
	"net/http"
	"time"
)

type RetryTransport struct {
	base       http.RoundTripper
	maxRetries int
	minWait    time.Duration
	maxWait    time.Duration
}

func NewRetryTransport(base http.RoundTripper, maxRetries int, minWait, maxWait time.Duration) *RetryTransport {
	return &RetryTransport{
		base:       base,
		maxRetries: maxRetries,
		minWait:    minWait,
		maxWait:    maxWait,
	}
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := t.base.RoundTrip(req)

		if attempt >= t.maxRetries || !t.shouldRetry(req.Context(), resp, err) {
			return resp, err
		}

		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(t.backoff(attempt)):
		}
	}
}

func (t *RetryTransport) shouldRetry(ctx context.Context, resp *http.Response, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	if err != nil {
		return true
	}
	return resp.StatusCode >= 500
}

func (t *RetryTransport) backoff(attempt int) time.Duration {
	d := t.minWait * (1 << attempt)
	if d > t.maxWait {
		return t.maxWait
	}
	return d
}
