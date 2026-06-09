package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrExpired          = errors.New("token has expired")
	ErrInvalidSignature = errors.New("token signature verification failed")
	ErrInvalidToken     = errors.New("invalid token")
)

type Claims struct {
	Subject   string
	Audience  []string
	Scopes    []string
	JTI       string
	IsAdmin   bool
	IssuedAt  time.Time
	ExpiresAt time.Time
	Issuer    string
}

// Validates a raw JWT string and returns its verified claims.
// Implementations must return ErrExpired, ErrInvalidSignature, or ErrInvalidToken
// for the respective failure modes so callers can distinguish them.
type Verifier interface {
	Verify(ctx context.Context, token string) (*Claims, error)
}
