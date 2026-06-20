package csrf

import (
	"net/http"

	"github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
	"github.com/rdevitto86/komodo-forge-sdk-go/security/token"
)

func GenerateToken() (string, error) {
	return token.Hex(32)
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
