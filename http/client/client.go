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

type ClientConfig struct {
	Timeout        time.Duration
	Transport      http.RoundTripper
	CircuitBreaker *BreakerConfig
}

type Client struct {
	httpClient *http.Client
	breaker    *breaker
}

// Returns a Client configured from cfg; zero-value cfg defaults to a 30s timeout, DefaultTransport, and no circuit breaker.
func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport := cfg.Transport
	if transport == nil {
		transport = DefaultTransport
	}

	c := &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}

	if cfg.CircuitBreaker != nil {
		c.breaker = newBreaker(*cfg.CircuitBreaker)
	}
	return c
}

// Executes the request; when a circuit breaker is configured, failures are counted per host and ErrOpen is returned when the breaker is open.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
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
		if resp.StatusCode >= 400 {
			return fmt.Errorf("upstream %d", resp.StatusCode)
		}
		return nil
	})

	if breakerErr == ErrOpen {
		return nil, ErrOpen
	}

	// For non-ErrOpen breaker errors (transport error or 4xx/5xx), return the
	// original resp and err so the caller sees the real HTTP response when present.
	return resp, err
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
		body, _ := io.ReadAll(res.Body)
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: body}
	}

	var result T
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
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
		raw, _ := io.ReadAll(res.Body)
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: raw}
	}

	var result T
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &result, nil
}
