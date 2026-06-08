package client

import (
	"errors"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxAttempts int                                       // total attempts including the first; recommended default 3
	BaseDelay   time.Duration                             // backoff before the first retry, doubling each attempt up to MaxDelay; recommended default 100ms
	MaxDelay    time.Duration                             // caps the exponential backoff between attempts; recommended default 2s
	ShouldRetry func(resp *http.Response, err error) bool // decides whether to retry; nil retries transport errors, 429, and 5xx
}

func defaultShouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500)
}

// Issues req through do, retrying with exponential backoff until ShouldRetry returns false,
// the attempt budget is exhausted, the request context is done, or the breaker reports open.
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	cfg := c.retry
	delay := cfg.BaseDelay

	var resp *http.Response
	var err error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		attemptReq := req
		if attempt > 1 {
			attemptReq, err = cloneRequest(req)
			if err != nil {
				return nil, err
			}
		}

		resp, err = c.do(attemptReq)
		if errors.Is(err, ErrOpen) || attempt == cfg.MaxAttempts || !cfg.ShouldRetry(resp, err) {
			return resp, err
		}

		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
		delay = min(delay*2, cfg.MaxDelay)
	}
	return resp, err
}

// Produces a replayable copy of req via GetBody, since the original body is consumed
// by the first attempt.
func cloneRequest(req *http.Request) (*http.Request, error) {
	clone := req.Clone(req.Context())
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		clone.Body = body
	}
	return clone, nil
}
