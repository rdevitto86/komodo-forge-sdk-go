// Package communications is the SDK adapter for komodo-communications-api.
// Types in this file are derived from komodo-communications-api/docs/openapi.yaml.
//
// The client exposes two layers:
//
//  1. Typed methods (e.g. SendOTP) for common high-level operations. Most consumers
//     should use these — they hide URL composition, template_id selection, and
//     response unmarshaling.
//
//  2. A Raw() escape hatch returning the underlying *httpc.Client. Use this when
//     calling a route the typed layer doesn't cover yet.
//
// Versioning: the API version is per-client, set at construction. To target a
// different version, construct a separate Client. Base URL is per-client as well —
// per-call URL override is intentionally not supported (see api/adapters/README.md).
package communications

import (
	"context"
	"fmt"

	httpc "github.com/rdevitto86/komodo-forge-sdk-go/http/client"
)

// supportedVersions enumerates the API versions this client knows how to talk to.
// Update this when komodo-communications-api ships a new major version and the
// typed surface here is verified against it.
var supportedVersions = map[int]struct{}{
	1: {},
}

// Client is the SDK adapter for komodo-communications-api.
type Client struct {
	baseURL string
	version int
	http    *httpc.Client
}

// NewClient constructs a Client targeting baseURL at API version ver.
// Returns an error if baseURL is empty or ver is not in supportedVersions.
func NewClient(baseURL string, ver int) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("communications: baseURL is required")
	}
	if _, ok := supportedVersions[ver]; !ok {
		return nil, fmt.Errorf("communications: unsupported api version %d", ver)
	}
	return &Client{
		baseURL: baseURL,
		version: ver,
		http:    httpc.NewClient(),
	}, nil
}

// Raw returns the underlying HTTP client for callers that need to compose
// requests for routes the typed surface doesn't expose. Use sparingly — prefer
// adding a typed method here so all consumers benefit.
func (c *Client) Raw() *httpc.Client { return c.http }

// url builds a fully-qualified URL for the configured version.
func (c *Client) url(path string) string {
	return fmt.Sprintf("%s/v%d%s", c.baseURL, c.version, path)
}

// --- Typed surface ---------------------------------------------------------

// SendEmailRequest matches the SendEmailRequest schema in
// komodo-communications-api/docs/openapi.yaml.
type SendEmailRequest struct {
	To           string         `json:"to"`
	TemplateID   string         `json:"template_id"`
	TemplateData map[string]any `json:"template_data,omitempty"`
}

// SendResult matches the SendResult schema in
// komodo-communications-api/docs/openapi.yaml.
type SendResult struct {
	MessageID string `json:"message_id"`
}

// SendEmail dispatches a templated email via communications-api.
// This is the generic primitive — prefer purpose-built helpers (SendOTP,
// SendOrderConfirmation, etc.) which encode the template_id contract.
func (c *Client) SendEmail(ctx context.Context, req SendEmailRequest) (*SendResult, error) {
	res, err := httpc.PostJSON[SendResult](c.http, ctx, c.url("/send/email"), req)
	if err != nil {
		return nil, fmt.Errorf("communications.SendEmail: %w", err)
	}
	return res, nil
}

// SendOTP dispatches a one-time password to email via the "otp-request" template.
// The template is expected to render `code` and `ttl_seconds` from template_data.
//
// Delivery semantics are the caller's concern: komodo-auth-api treats delivery
// failure as non-fatal (logs + returns 200) to prevent email-existence probing.
// This method just reports the underlying HTTP outcome.
func (c *Client) SendOTP(ctx context.Context, email, code string, ttlSeconds int64) error {
	_, err := c.SendEmail(ctx, SendEmailRequest{
		To:         email,
		TemplateID: "otp-request",
		TemplateData: map[string]any{
			"code":        code,
			"ttl_seconds": ttlSeconds,
		},
	})
	if err != nil {
		return fmt.Errorf("communications.SendOTP: %w", err)
	}
	return nil
}
