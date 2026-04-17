package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps net/http.Client as the canonical HTTP transport for Komodo services.
// Future additions: timeout override, retry policy, circuit breaker.
type Client struct {
	httpClient *http.Client
}

// NewClient returns a Client with a 30s default timeout.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Do executes the request using the underlying http.Client.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
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

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.GetJSON: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("client.GetJSON: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: body}
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.PostJSON: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("client.PostJSON: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: raw}
	}

	var result T
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("client.PostJSON: unmarshal: %w", err)
	}
	return &result, nil
}
