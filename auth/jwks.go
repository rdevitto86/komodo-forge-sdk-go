package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Config holds the options for constructing a JWKSVerifier.
type Config struct {
	JWKSURL    string
	CacheTTL   time.Duration
	HTTPClient *http.Client
}

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksDoc struct {
	Keys []jwk `json:"keys"`
}

type jwtClaims struct {
	Scopes  []string `json:"scp,omitempty"`
	IsAdmin bool     `json:"adm,omitempty"`
	gojwt.RegisteredClaims
}

// JWKSVerifier implements Verifier by resolving RS256 public keys from a remote JWKS endpoint.
// Keys are cached by kid; a cache miss triggers a single re-fetch before returning ErrInvalidToken.
type JWKSVerifier struct {
	cfg   Config
	mu    sync.RWMutex
	cache map[string]*rsa.PublicKey
}

// Returns a JWKSVerifier that fetches public keys from cfg.JWKSURL and caches them by kid.
// Returns an error if cfg.JWKSURL is empty.
func NewJWKSVerifier(cfg Config) (*JWKSVerifier, error) {
	if cfg.JWKSURL == "" {
		return nil, errors.New("invalid JWKS configuration: JWKSURL is required")
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &JWKSVerifier{
		cfg:   cfg,
		cache: make(map[string]*rsa.PublicKey),
	}, nil
}

// Validates a raw JWT string against keys fetched from the configured JWKS endpoint.
// Returns ErrExpired, ErrInvalidSignature, or ErrInvalidToken for the respective failure modes.
func (v *JWKSVerifier) Verify(ctx context.Context, token string) (*Claims, error) {
	kid, err := extractKID(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	key, err := v.resolveKey(ctx, kid, false)
	if err != nil {
		return nil, err
	}
	return v.verifyWithKey(token, key)
}

func (v *JWKSVerifier) resolveKey(ctx context.Context, kid string, afterRefresh bool) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.cache[kid]
	v.mu.RUnlock()

	if ok {
		return key, nil
	}
	if afterRefresh {
		return nil, ErrInvalidToken
	}
	if err := v.refreshCache(ctx); err != nil {
		return nil, err
	}
	return v.resolveKey(ctx, kid, true)
}

func (v *JWKSVerifier) refreshCache(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build JWKS request: %w", err)
	}

	res, err := v.cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: unexpected status %d", res.StatusCode)
	}

	var doc jwksDoc
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return fmt.Errorf("failed to parse JWKS response: %w", err)
	}

	newCache := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Use != "sig" {
			continue
		}
		pub, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			return fmt.Errorf("failed to parse public key for kid %q: %w", k.Kid, err)
		}
		newCache[k.Kid] = pub
	}

	v.mu.Lock()
	v.cache = newCache
	v.mu.Unlock()

	return nil
}

func (v *JWKSVerifier) verifyWithKey(tokenString string, key *rsa.PublicKey) (*Claims, error) {
	c := &jwtClaims{}
	parsed, err := gojwt.ParseWithClaims(tokenString, c, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("invalid signing method: %v", t.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		if errors.Is(err, gojwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		if errors.Is(err, gojwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, ErrInvalidToken
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*jwtClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return mapClaims(claims), nil
}

func extractKID(tokenString string) (string, error) {
	p := gojwt.NewParser()
	tok, _, err := p.ParseUnverified(tokenString, &gojwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token header: %w", err)
	}
	kid, _ := tok.Header["kid"].(string)
	return kid, nil
}

func parseRSAPublicKey(n, e string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}
	nInt := new(big.Int).SetBytes(nBytes)
	eInt := int(new(big.Int).SetBytes(eBytes).Int64())
	return &rsa.PublicKey{N: nInt, E: eInt}, nil
}

func mapClaims(c *jwtClaims) *Claims {
	cl := &Claims{
		Subject:  c.Subject,
		Audience: []string(c.Audience),
		Scopes:   c.Scopes,
		JTI:      c.ID,
		IsAdmin:  c.IsAdmin,
		Issuer:   c.Issuer,
	}
	if c.IssuedAt != nil {
		cl.IssuedAt = c.IssuedAt.Time
	}
	if c.ExpiresAt != nil {
		cl.ExpiresAt = c.ExpiresAt.Time
	}
	return cl
}
