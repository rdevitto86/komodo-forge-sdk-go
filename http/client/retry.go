package client

import (
	"errors"
	"net/http"
	"time"
)

// Controls retry-with-backoff behavior attached via ClientConfig.Retry.
type RetryConfig struct {
	// MaxAttempts is the total number of attempts, including the first.
	// Recommended default: 3.
	MaxAttempts int

	// BaseDelay is the backoff before the first retry; it doubles after each
	// subsequent attempt up to MaxDelay.
	// Recommended default: 100ms.
	BaseDelay time.Duration

	// MaxDelay caps the exponential backoff between attempts.
	// Recommended default: 2s.
	MaxDelay time.Duration

	// ShouldRetry decides whether a response/error warrants another attempt.
	// nil falls back to retrying transport errors, 429, and 5xx responses.
	ShouldRetry func(resp *http.Response, err error) bool
}

func defaultShouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500)
}

// doWithRetry issues req through do, retrying with exponential backoff until cfg.ShouldRetry
// returns false, the attempt budget is exhausted, the request context is done, or the
// circuit breaker reports it is open.
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

// cloneRequest produces a replayable copy of req via GetBody, since the original
// request body is consumed by the first attempt.
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
