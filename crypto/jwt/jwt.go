// Package jwt re-exports the security/jwt package at the crypto/jwt import path.
// Services import this package as: "github.com/rdevitto86/komodo-forge-sdk-go/crypto/jwt"
package jwt

import (
	"net/http"

	securejwt "github.com/rdevitto86/komodo-forge-sdk-go/security/jwt"
)

type CustomClaims = securejwt.CustomClaims

// Loads JWT signing and verification keys from environment variables.
func InitializeKeys() error {
	return securejwt.InitializeKeys()
}

// Mints a signed JWT with the given claims.
func SignToken(issuer, subject, audience string, ttl int64, scopes []string) (string, error) {
	return securejwt.SignToken(issuer, subject, audience, ttl, scopes)
}

// Parses and validates the token string, returning true if valid.
func ValidateToken(tokenString string) (bool, error) {
	return securejwt.ValidateToken(tokenString)
}

// Validates the token and returns its claims in a single parse.
// Prefer this over ValidateToken + ParseClaims.
func ValidateAndParseClaims(tokenString string) (*CustomClaims, error) {
	return securejwt.ValidateAndParseClaims(tokenString)
}

// Parses the token string and returns the embedded CustomClaims.
func ParseClaims(tokenString string) (*CustomClaims, error) {
	return securejwt.ParseClaims(tokenString)
}

// Extracts the Bearer token from the Authorization header.
func ExtractTokenFromRequest(req *http.Request) (string, error) {
	return securejwt.ExtractTokenFromRequest(req)
}
