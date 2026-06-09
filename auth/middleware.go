package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
	"github.com/rdevitto86/komodo-forge-sdk-go/security/jwt"
)

// Validates the Bearer JWT, injects auth claims into the request context, and rejects invalid tokens.
// Deprecated: use Middleware with an injected Verifier for testability.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		tokenString, err := jwt.ExtractTokenFromRequest(req)
		if err != nil {
			logger.Error("failed to extract token", err)
			httpErr.SendError(wtr, req, httpErr.Auth.InvalidToken, httpErr.WithDetail("missing or malformed authorization header"))
			return
		}

		claims, err := jwt.ValidateAndParseClaims(tokenString)
		if err != nil {
			logger.Error("token validation failed", err)
			httpErr.SendError(wtr, req, httpErr.Auth.InvalidToken, httpErr.WithDetail("token validation failed"))
			return
		}

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

// Validates the Bearer JWT using v and injects verified claims into the request context.
// Prefer this over AuthMiddleware — it accepts an injected Verifier for testability.
func Middleware(v Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
			tokenString, err := jwt.ExtractTokenFromRequest(req)
			if err != nil {
				logger.Error("failed to extract token from request", err)
				httpErr.SendError(wtr, req, httpErr.Auth.InvalidToken, httpErr.WithDetail("missing or malformed authorization header"))
				return
			}

			claims, err := v.Verify(req.Context(), tokenString)
			if err != nil {
				logger.Error("token verification failed", err)
				errCode := httpErr.Auth.InvalidToken
				if errors.Is(err, ErrExpired) {
					errCode = httpErr.Auth.ExpiredToken
				}
				httpErr.SendError(wtr, req, errCode, httpErr.WithDetail("token verification failed"))
				return
			}

			ctx := context.WithValue(req.Context(), ctxKeys.AUTH_VALID_KEY, true)

			if claims.Subject != "" {
				ctx = context.WithValue(ctx, ctxKeys.USER_ID_KEY, claims.Subject)
			}
			if claims.JTI != "" {
				ctx = context.WithValue(ctx, ctxKeys.SESSION_ID_KEY, claims.JTI)
			}

			isUIRequest := len(claims.Scopes) == 0 || claims.IsAdmin
			isAPIRequest := len(claims.Scopes) > 0

			if isUIRequest {
				ctx = context.WithValue(ctx, ctxKeys.REQUEST_TYPE_KEY, "ui")
				ctx = context.WithValue(ctx, ctxKeys.CLIENT_TYPE_KEY, "browser")
			}
			if isAPIRequest {
				ctx = context.WithValue(ctx, ctxKeys.REQUEST_TYPE_KEY, "api")
				ctx = context.WithValue(ctx, ctxKeys.SCOPES_KEY, claims.Scopes)
				ctx = context.WithValue(ctx, ctxKeys.CLIENT_TYPE_KEY, "api")
			}
			if claims.IsAdmin {
				ctx = context.WithValue(ctx, ctxKeys.IS_ADMIN_KEY, true)
			}

			next.ServeHTTP(wtr, req.WithContext(ctx))
		})
	}
}

// Rejects requests whose JWT does not carry at least one "svc:"-prefixed scope.
//
// Service identity is conveyed by a "svc:<name>" scope: the Auth API issues machine tokens
// (client_credentials grant) carrying such a scope, and consumers obtain them via
// http/client.WithServiceAuth. Tokens should also set aud to the target service for
// defense-in-depth, enforced separately by auth.JWKSVerifier (ExpectedAudience).
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
