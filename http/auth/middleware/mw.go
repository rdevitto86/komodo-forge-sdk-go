package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/rdevitto86/komodo-forge-sdk-go/crypto/jwt"
	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/http/errors"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

// Handles OAuth2 + JWT Bearer token authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		tokenString, err := jwt.ExtractTokenFromRequest(req)
		if err != nil {
			logger.Error("failed to extract token", err)
			httpErr.SendError(wtr, req, httpErr.Auth.InvalidToken, httpErr.WithDetail(err.Error()))
			return
		}

		valid, err := jwt.ValidateToken(tokenString)
		if !valid || err != nil {
			logger.Error("token validation failed", err)
			httpErr.SendError(wtr, req, httpErr.Auth.InvalidToken, httpErr.WithDetail(err.Error()))
			return
		}

		// ParseClaims is called after ValidateToken confirms the signature, issuer,
		// and audience are valid; an error here would be unreachable in practice.
		claims, _ := jwt.ParseClaims(tokenString)

		ctx := context.WithValue(req.Context(), ctxKeys.AUTH_VALID_KEY, true)
		
		if claims.Subject != "" {
			ctx = context.WithValue(ctx, ctxKeys.USER_ID_KEY, claims.Subject)
		}
		if claims.ID != "" {
			ctx = context.WithValue(ctx, ctxKeys.SESSION_ID_KEY, claims.ID)
		}

		isUIRequest := len(claims.Scopes) == 0 || claims.IsAdmin
		isAPIRequest := len(claims.Scopes) > 0

		if isUIRequest {
			ctx = context.WithValue(ctx, ctxKeys.REQUEST_TYPE_KEY, "ui")
		}
		if isAPIRequest {
			ctx = context.WithValue(ctx, ctxKeys.REQUEST_TYPE_KEY, "api")
			ctx = context.WithValue(ctx, ctxKeys.SCOPES_KEY, claims.Scopes)
		}
		if claims.IsAdmin {
			ctx = context.WithValue(ctx, ctxKeys.IS_ADMIN_KEY, true)
		}

		next.ServeHTTP(wtr, req.WithContext(ctx))
	})
}

// RequireServiceScope rejects requests whose JWT does not carry a service-scoped token.
// Service tokens must have at least one scope with a "svc:" prefix, issued by komodo-auth-api
// on service bootstrap — distinct from user tokens, which carry user-scoped claims.
func RequireServiceScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		scopes, ok := req.Context().Value(ctxKeys.SCOPES_KEY).([]string)
		if !ok || len(scopes) == 0 {
			httpErr.SendError(wtr, req, httpErr.Auth.InsufficientScope)
			return
		}

		for _, s := range scopes {
			if strings.HasPrefix(s, "svc:") {
				next.ServeHTTP(wtr, req)
				return
			}
		}

		httpErr.SendError(wtr, req, httpErr.Auth.InsufficientScope)
	})
}
