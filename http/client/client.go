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

// DefaultTransport is a tuned *http.Transport used by NewClient when no
// WithTransport option is supplied. Callers that need TLS customisation or
// proxy support should supply their own via WithTransport.
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

// Client wraps net/http.Client as the canonical HTTP transport for Komodo services.
type Client struct {
	httpClient *http.Client
	breaker    *breaker
}

// Option is a functional option for NewClient.
type Option func(*Client)

// WithCircuitBreaker attaches a circuit breaker to the client. When set, Do()
// tracks failures (transport errors and 4xx/5xx responses) per request host.
func WithCircuitBreaker(cfg Config) Option {
	return func(c *Client) {
		c.breaker = newBreaker(cfg)
	}
}

// WithTransport replaces the underlying http.RoundTripper. Use this to supply
// a custom TLS config, proxy, or test transport.
func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) {
		c.httpClient.Transport = t
	}
}

// NewClient returns a Client with a 30s default timeout and DefaultTransport.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: DefaultTransport,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Do executes the request using the underlying http.Client.
// When a circuit breaker is configured, failures (transport errors and 4xx/5xx
// responses) are counted per req.URL.Host. If the breaker is open, Do returns
// nil, ErrOpen without making a network call.
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

// HTTPError is returned by GetJSON and PostJSON when the server responds with a non-2xx status.
// Callers can errors.As to inspect the status code (e.g. distinguish 409 conflict from 503).
type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("upstream returned %d: %s", e.StatusCode, e.Body)
}

// GetJSON issues a GET to url with context, and JSON-decodes the response body into T.
// Returns *HTTPError for any non-2xx response.
func GetJSON[T any](c *Client, ctx context.Context, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("client.GetJSON: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.GetJSON: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("client.GetJSON: read body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: body}
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("client.GetJSON: unmarshal: %w", err)
	}
	return &result, nil
}

// PostJSON marshals body as JSON, issues a POST to url with context, and decodes a 2xx response into T.
// Returns *HTTPError for any non-2xx response.
func PostJSON[T any](c *Client, ctx context.Context, url string, body any) (*T, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("client.PostJSON: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("client.PostJSON: %w", err)
	}
	// Set GetBody so the request can be replayed by redirects or retry logic.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.PostJSON: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("client.PostJSON: read body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: raw}
	}

	var result T
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("client.PostJSON: unmarshal: %w", err)
	}
	return &result, nil
}
