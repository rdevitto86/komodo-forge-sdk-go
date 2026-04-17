// Package jwt re-exports the security/jwt package at the crypto/jwt import path.
// Services import this package as: "github.com/rdevitto86/komodo-forge-sdk-go/crypto/jwt"
package jwt

import (
	"net/http"

	securejwt "github.com/rdevitto86/komodo-forge-sdk-go/security/jwt"
)

// CustomClaims defines type-safe JWT claims.
type CustomClaims = securejwt.CustomClaims

// InitializeKeys loads JWT signing and verification keys from environment variables.
func InitializeKeys() error {
	return securejwt.InitializeKeys()
}

// SignToken mints a signed JWT with the given claims.
func SignToken(issuer, subject, audience string, ttl int64, scopes []string) (string, error) {
	return securejwt.SignToken(issuer, subject, audience, ttl, scopes)
}

// ValidateToken parses and validates the token string, returning true if valid.
func ValidateToken(tokenString string) (bool, error) {
	return securejwt.ValidateToken(tokenString)
}

// ParseClaims parses the token string and returns the embedded CustomClaims.
func ParseClaims(tokenString string) (*CustomClaims, error) {
	return securejwt.ParseClaims(tokenString)
}

// ExtractTokenFromRequest extracts the Bearer token from the Authorization header.
func ExtractTokenFromRequest(req *http.Request) (string, error) {
	return securejwt.ExtractTokenFromRequest(req)
}
