package client

import (
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
