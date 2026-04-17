package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

func NewClientWithBaseURL(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *Client) Get(path string) (*http.Response, error) {
	return c.httpClient.Get(c.baseURL + path)
}

func (c *Client) Post(path string, body []byte) (*http.Response, error) {
	return c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewBuffer(body))
}

func (c *Client) Put(path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("PUT", c.baseURL+path, bytes.NewBuffer(body))
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) Delete(path string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil { return nil, err }
	return c.httpClient.Do(req)
}

func (c *Client) Patch(path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("PATCH", c.baseURL+path, bytes.NewBuffer(body))
	if err != nil { return nil, err }
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

// GetJSON performs a GET request with context and unmarshals the JSON response into v.
func (c *Client) GetJSON(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return &httpError{
			StatusCode: resp.StatusCode,
			Body:       body,
		}
	}

	return json.Unmarshal(body, v)
}

// httpError represents an HTTP error response.
type httpError struct {
	StatusCode int
	Body       []byte
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}
