// Package support is the SDK adapter for komodo-support-api.
// Types in this file are derived from komodo-support-api/docs/openapi.yaml.
//
// The client exposes two layers:
//
//  1. Typed methods for common high-level operations. Most consumers should use
//     these — they hide URL composition and response unmarshaling.
//
//  2. A Raw() escape hatch returning the underlying *httpc.Client. Use this when
//     calling a route the typed layer doesn't cover yet.
//
// Versioning: the API version is per-client, set at construction. To target a
// different version, construct a separate Client. Base URL is per-client as well —
// per-call URL override is intentionally not supported (see api/adapters/README.md).
package support

import (
	"fmt"

	httpc "github.com/rdevitto86/komodo-forge-sdk-go/http/client"
)

// supportedVersions enumerates the API versions this client knows how to talk to.
// Update this when komodo-support-api ships a new major version and the typed
// surface here is verified against it.
var supportedVersions = map[int]struct{}{
	1: {},
}

// Client is the SDK adapter for komodo-support-api.
type Client struct {
	baseURL string
	version int
	http    *httpc.Client
}

// NewClient constructs a Client targeting baseURL at API version ver.
// Returns an error if baseURL is empty or ver is not in supportedVersions.
func NewClient(baseURL string, ver int) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("support: baseURL is required")
	}
	if _, ok := supportedVersions[ver]; !ok {
		return nil, fmt.Errorf("support: unsupported api version %d", ver)
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

// --- Typed surface ---------------------------------------------------------
