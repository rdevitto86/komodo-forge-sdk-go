package csrf

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
)

// Generates a random hex-encoded CSRF token for the double-submit cookie pattern.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate csrf token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func setCookie(wtr http.ResponseWriter, token string) {
	http.SetCookie(wtr, &http.Cookie{
		Name:     headers.COOKIE_CSRF_TOKEN,
		Value:    token,
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// Returns the request's CSRF cookie value, minting and setting one when absent so the client can echo it back.
func ensureToken(wtr http.ResponseWriter, req *http.Request) (string, error) {
	if cookie, err := req.Cookie(headers.COOKIE_CSRF_TOKEN); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	token, err := GenerateToken()
	if err != nil {
		return "", err
	}
	setCookie(wtr, token)
	return token, nil
}
