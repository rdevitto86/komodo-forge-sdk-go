package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var DefaultTransport http.RoundTripper = &http.Transport{
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
}

// DefaultMaxResponseBytes caps how much of a response body GetJSON/PostJSON will read
// when ClientConfig.MaxResponseBytes is left at zero, guarding against memory exhaustion
// from a hostile or buggy upstream. Set MaxResponseBytes < 0 to disable the cap.
const DefaultMaxResponseBytes int64 = 10 << 20 // 10 MiB

type ClientConfig struct {
	Timeout          time.Duration
	Transport        http.RoundTripper
	CircuitBreaker   *BreakerConfig
	Retry            *RetryConfig
	MaxResponseBytes int64 // per-response read cap; 0 = DefaultMaxResponseBytes, <0 = unlimited
}

type Client struct {
	httpClient   *http.Client
	breaker      *breaker
	retry        *RetryConfig
	maxRespBytes int64
}

// Returns a Client configured from cfg; zero-value cfg defaults to a 30s timeout, DefaultTransport, and no circuit breaker or retry.
func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport := cfg.Transport
	if transport == nil {
		transport = DefaultTransport
	}

	maxResp := cfg.MaxResponseBytes
	if maxResp == 0 {
		maxResp = DefaultMaxResponseBytes
	}

	c := &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		maxRespBytes: maxResp,
	}

	if cfg.CircuitBreaker != nil {
		c.breaker = newBreaker(*cfg.CircuitBreaker)
	}

	if cfg.Retry != nil {
		r := *cfg.Retry
		if r.MaxAttempts <= 0 {
			r.MaxAttempts = 3
		}
		if r.BaseDelay <= 0 {
			r.BaseDelay = 100 * time.Millisecond
		}
		if r.MaxDelay <= 0 {
			r.MaxDelay = 2 * time.Second
		}
		if r.ShouldRetry == nil {
			r.ShouldRetry = defaultShouldRetry
		}
		c.retry = &r
	}
	return c
}

// Executes the request, retrying with backoff when configured and routing each attempt through the circuit breaker;
// failures are counted per host and ErrOpen is returned when the breaker is open.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.retry != nil {
		return c.doWithRetry(req)
	}
	return c.do(req)
}

// Issues a single attempt, routing through the circuit breaker when one is configured.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.breaker == nil {
		return c.httpClient.Do(req)
	}

	var (
		resp *http.Response
		err  error
	)

	breakerErr := c.breaker.execute(req.URL.Host, func() error {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return err
		}
		// Count only 5xx as upstream failures. A 4xx reflects a caller mistake
		// (bad request, not found, unauthorized), not service health, so it must
		// not trip the breaker.
		if resp.StatusCode >= 500 {
			return fmt.Errorf("upstream returned %d", resp.StatusCode)
		}
		return nil
	})

	if breakerErr == ErrOpen {
		return nil, ErrOpen
	}

	// For non-ErrOpen breaker errors (transport error or 5xx), return the
	// original resp and err so the caller sees the real HTTP response when present.
	return resp, err
}

// limitedBody wraps r so reads stop after the configured MaxResponseBytes, preventing a
// hostile upstream from forcing unbounded allocation. A negative cap disables the limit.
func (c *Client) limitedBody(r io.Reader) io.Reader {
	if c.maxRespBytes < 0 {
		return r
	}
	return io.LimitReader(r, c.maxRespBytes)
}

type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("upstream returned %d: %s", e.StatusCode, e.Body)
}

// Issues a GET and JSON-decodes a 2xx response body into T; returns *HTTPError for non-2xx responses.
func GetJSON[T any](c *Client, ctx context.Context, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(c.limitedBody(res.Body))
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: body}
	}

	var result T
	if err := json.NewDecoder(c.limitedBody(res.Body)).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &result, nil
}

// Marshals body as JSON, POSTs to url, and decodes a 2xx response into T; returns *HTTPError for non-2xx responses.
func PostJSON[T any](c *Client, ctx context.Context, url string, body any) (*T, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	// Set GetBody so the request can be replayed by redirects or retry logic.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		raw, _ := io.ReadAll(c.limitedBody(res.Body))
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: raw}
	}

	var result T
	if err := json.NewDecoder(c.limitedBody(res.Body)).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &result, nil
}
