package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultTokenTTL is the cache lifetime applied to a fetched service token when the
// token endpoint omits expires_in, bounding how often a missing-TTL response is re-fetched.
const DefaultTokenTTL = 5 * time.Minute

// defaultRefreshFraction is the portion of a token's lifetime that must remain before the
// cached token is treated as stale, so a refresh happens proactively rather than on expiry.
const defaultRefreshFraction = 0.15

// ServiceAuthConfig configures a ClientCredentialsTokenSource that obtains service tokens
// from the central Auth API via the OAuth2 client_credentials grant. It holds no signing
// key — issuance stays with the Auth API; this only presents credentials and caches the result.
type ServiceAuthConfig struct {
	// TokenURL is the Auth API token endpoint (e.g. https://auth.internal/v1/oauth/token).
	TokenURL string
	// ClientID and ClientSecret are the calling service's machine credentials.
	ClientID     string
	ClientSecret string
	// Scope is the space-separated scope set requested for the token (e.g. "svc:order-api").
	Scope string
	// HTTPClient performs the token request; DefaultTransport-backed client when nil.
	HTTPClient *http.Client
}

// tokenResponse is the standard OAuth2 client_credentials success body.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ClientCredentialsTokenSource fetches, caches, and proactively refreshes a service token
// from the Auth API. It is safe for concurrent use; concurrent refreshes collapse to one fetch.
type ClientCredentialsTokenSource struct {
	cfg        ServiceAuthConfig
	httpClient *http.Client
	sf         singleflight.Group

	mu        sync.RWMutex
	token     string
	refreshAt time.Time
}

// NewClientCredentialsTokenSource validates cfg and returns a token source; it errors when the
// token URL or client credentials are missing.
func NewClientCredentialsTokenSource(cfg ServiceAuthConfig) (*ClientCredentialsTokenSource, error) {
	if cfg.TokenURL == "" {
		return nil, fmt.Errorf("service auth: TokenURL is required")
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("service auth: ClientID and ClientSecret are required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second, Transport: DefaultTransport}
	}

	return &ClientCredentialsTokenSource{cfg: cfg, httpClient: httpClient}, nil
}

// Token returns a valid service token, fetching or refreshing from the Auth API when the cached
// token is absent or within its refresh window. Concurrent callers share a single in-flight fetch.
func (s *ClientCredentialsTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.RLock()
	tok, refreshAt := s.token, s.refreshAt
	s.mu.RUnlock()
	if tok != "" && time.Now().Before(refreshAt) {
		return tok, nil
	}

	v, err, _ := s.sf.Do("token", func() (any, error) {
		// Re-check under the singleflight: a prior caller may have just refreshed.
		s.mu.RLock()
		tok, refreshAt := s.token, s.refreshAt
		s.mu.RUnlock()
		if tok != "" && time.Now().Before(refreshAt) {
			return tok, nil
		}
		return s.fetch(ctx)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// fetch performs the client_credentials request and stores the resulting token with its refresh window.
func (s *ClientCredentialsTokenSource) fetch(ctx context.Context) (string, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
	}
	if s.cfg.Scope != "" {
		form.Set("scope", s.cfg.Scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("service auth: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("service auth: token request failed: %w", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, DefaultMaxResponseBytes))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("service auth: token endpoint returned %d", res.StatusCode)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("service auth: decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("service auth: token response missing access_token")
	}

	ttl := DefaultTokenTTL
	if tr.ExpiresIn > 0 {
		ttl = time.Duration(tr.ExpiresIn) * time.Second
	}

	s.mu.Lock()
	s.token = tr.AccessToken
	s.refreshAt = time.Now().Add(ttl - time.Duration(float64(ttl)*defaultRefreshFraction))
	s.mu.Unlock()

	return tr.AccessToken, nil
}

// authTransport attaches a service Bearer token, fetched from src, to every outbound request.
type authTransport struct {
	base http.RoundTripper
	src  *ClientCredentialsTokenSource
}

// RoundTrip clones req, sets a fresh Authorization Bearer header from the token source, and delegates.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.src.Token(req.Context())
	if err != nil {
		return nil, err
	}
	// Per the RoundTripper contract the inbound request must not be mutated.
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+token)
	return t.base.RoundTrip(clone)
}

// WithServiceAuth wraps base so every request carries a service Bearer token obtained from src via
// the Auth API client_credentials grant; a nil base uses DefaultTransport. Compose it into
// ClientConfig.Transport (or any generated client's transport) to authenticate service-to-service calls.
func WithServiceAuth(base http.RoundTripper, src *ClientCredentialsTokenSource) http.RoundTripper {
	if base == nil {
		base = DefaultTransport
	}
	return &authTransport{base: base, src: src}
}
