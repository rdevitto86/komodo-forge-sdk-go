package jwt

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	JWT_PUBLIC_KEY  = "JWT_PUBLIC_KEY"
	JWT_PRIVATE_KEY = "JWT_PRIVATE_KEY"
	JWT_AUDIENCE    = "JWT_AUDIENCE"
	JWT_ISSUER      = "JWT_ISSUER"
	JWT_KID         = "JWT_KID"
)

const defaultLeeway = 30 * time.Second

type SecretsProvider interface {
	GetSecret(ctx context.Context, name string) (string, error)
}

type CustomClaims struct {
	Scopes  []string `json:"scp,omitempty"`
	IsAdmin bool     `json:"adm,omitempty"`
	jwt.RegisteredClaims
}

type Config struct {
	PrivateKeyPEM  string
	PublicKeyPEM   string
	KID            string
	Issuer         string
	Audience       string
	Leeway         time.Duration
	Secrets        SecretsProvider
	PrivateKeyName string
}

type Client struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
	iss        string
	aud        string
	leeway     time.Duration
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.PublicKeyPEM == "" {
		return nil, fmt.Errorf("missing public key")
	}
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("missing issuer")
	}
	if cfg.Audience == "" {
		return nil, fmt.Errorf("missing audience")
	}

	pub, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cfg.PublicKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	privPEM := cfg.PrivateKeyPEM
	if privPEM == "" && cfg.Secrets != nil && cfg.PrivateKeyName != "" {
		privPEM, err = cfg.Secrets.GetSecret(ctx, cfg.PrivateKeyName)
		if err != nil {
			return nil, fmt.Errorf("failed to load private key from secrets: %w", err)
		}
	}

	var priv *rsa.PrivateKey
	if privPEM != "" {
		priv, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(privPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		if !priv.PublicKey.Equal(pub) {
			return nil, fmt.Errorf("private and public keys do not match")
		}
	}

	leeway := cfg.Leeway
	if leeway == 0 {
		leeway = defaultLeeway
	}

	return &Client{
		privateKey: priv,
		publicKey:  pub,
		kid:        cfg.KID,
		iss:        cfg.Issuer,
		aud:        cfg.Audience,
		leeway:     leeway,
	}, nil
}

func (c *Client) SignToken(issuer string, subject string, audience string, ttl int64, scopes []string) (string, error) {
	if c.privateKey == nil {
		return "", fmt.Errorf("failed to sign token: no private key configured")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("failed to sign token: ttl must be positive")
	}

	now := time.Now()
	claims := CustomClaims{
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttl) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = c.kid
	return token.SignedString(c.privateKey)
}

func (c *Client) ValidateToken(tokenString string) (bool, error) {
	token, _, err := c.parse(tokenString, true)
	if err != nil {
		return false, err
	}
	return token.Valid, nil
}

func (c *Client) ValidateAndParseClaims(tokenString string) (*CustomClaims, error) {
	_, claims, err := c.parse(tokenString, true)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (c *Client) ParseClaims(tokenString string) (*CustomClaims, error) {
	_, claims, err := c.parse(tokenString, false)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (c *Client) parse(tokenString string, checkIssAud bool) (*jwt.Token, *CustomClaims, error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(c.leeway),
	}
	if checkIssAud {
		opts = append(opts, jwt.WithIssuer(c.iss), jwt.WithAudience(c.aud))
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, c.keyFunc, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, nil, fmt.Errorf("failed to parse claims: unexpected claims type")
	}
	return token, claims, nil
}

func (c *Client) keyFunc(t *jwt.Token) (any, error) {
	if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
	}
	return c.publicKey, nil
}

func ExtractTokenFromRequest(req *http.Request) (string, error) {
	const prefix = "bearer "
	auth := strings.TrimSpace(req.Header.Get("Authorization"))
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return "", fmt.Errorf("missing or invalid authorization header")
	}
	return strings.TrimSpace(auth[len(prefix):]), nil
}
