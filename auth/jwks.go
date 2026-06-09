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
	"golang.org/x/sync/singleflight"
)

type Config struct {
	JWKSURL          string
	ExpectedAudience string // required; tokens must carry this audience
	ExpectedIssuer   string // required; tokens must carry this issuer
	CacheTTL         time.Duration
	HTTPClient       *http.Client
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

// negCacheTTL bounds how long an absent kid is remembered so repeat lookups of the
// same unknown kid don't re-hit the JWKS endpoint (DoS amplification guard).
const negCacheTTL = 30 * time.Second

// Resolves RS256 public keys from a remote JWKS endpoint to satisfy Verifier.
// Keys are cached by kid; a cache miss triggers a single, singleflight-deduped re-fetch
// before returning ErrInvalidToken, and tokens are checked against the configured
// audience and issuer.
type JWKSVerifier struct {
	cfg         Config
	mu          sync.RWMutex
	cache       map[string]*rsa.PublicKey
	negCache    map[string]time.Time // kid → time it was last confirmed absent
	lastRefresh time.Time
	sf          singleflight.Group
}

// Returns a JWKSVerifier that fetches public keys from cfg.JWKSURL and caches them by kid.
// Returns an error if JWKSURL, ExpectedAudience, or ExpectedIssuer is empty.
func NewJWKSVerifier(cfg Config) (*JWKSVerifier, error) {
	if cfg.JWKSURL == "" {
		return nil, errors.New("invalid JWKS configuration: JWKSURL is required")
	}
	if cfg.ExpectedAudience == "" {
		return nil, errors.New("invalid JWKS configuration: ExpectedAudience is required")
	}
	if cfg.ExpectedIssuer == "" {
		return nil, errors.New("invalid JWKS configuration: ExpectedIssuer is required")
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &JWKSVerifier{
		cfg:      cfg,
		cache:    make(map[string]*rsa.PublicKey),
		negCache: make(map[string]time.Time),
	}, nil
}

// Validates a raw JWT string against keys fetched from the configured JWKS endpoint,
// enforcing the configured audience and issuer.
// Returns ErrExpired, ErrInvalidSignature, or ErrInvalidToken for the respective failure modes.
func (v *JWKSVerifier) Verify(ctx context.Context, token string) (*Claims, error) {
	kid, err := extractKID(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	key, err := v.resolveKey(ctx, kid)
	if err != nil {
		return nil, err
	}
	return v.verifyWithKey(token, key)
}

func (v *JWKSVerifier) resolveKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.cache[kid]
	fresh := time.Since(v.lastRefresh) <= v.cfg.CacheTTL
	negAt, negative := v.negCache[kid]
	v.mu.RUnlock()

	// Fast path: key present and the cache is within its TTL.
	if ok && fresh {
		return key, nil
	}
	// Absent kid we recently confirmed missing — reject without hammering the endpoint.
	if !ok && negative && time.Since(negAt) < negCacheTTL {
		return nil, ErrInvalidToken
	}

	// Refresh (deduped via singleflight). On failure, fall back to a usable cached key.
	if err := v.refreshCache(ctx); err != nil {
		if ok {
			return key, nil
		}
		return nil, err
	}

	v.mu.RLock()
	key, ok = v.cache[kid]
	v.mu.RUnlock()
	if ok {
		return key, nil
	}

	v.mu.Lock()
	v.negCache[kid] = time.Now()
	v.mu.Unlock()
	return nil, ErrInvalidToken
}

// refreshCache collapses concurrent refreshes into a single upstream fetch.
func (v *JWKSVerifier) refreshCache(ctx context.Context) error {
	_, err, _ := v.sf.Do("refresh", func() (any, error) {
		return nil, v.fetchAndStore(ctx)
	})
	return err
}

func (v *JWKSVerifier) fetchAndStore(ctx context.Context) error {
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
	v.lastRefresh = time.Now()
	v.negCache = make(map[string]time.Time) // a rotated kid may now be present
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
	},
		gojwt.WithAudience(v.cfg.ExpectedAudience),
		gojwt.WithIssuer(v.cfg.ExpectedIssuer),
		gojwt.WithValidMethods([]string{"RS256"}),
	)

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
