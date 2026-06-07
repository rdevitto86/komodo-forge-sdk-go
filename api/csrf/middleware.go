package csrf

import (
	"context"
	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	"github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
	"net/http"
)

// Issues a double-submit CSRF cookie on every request and enforces a matching X-CSRF-Token
// header on state-changing requests from browser clients; verified API clients are exempt.
// Client type is read solely from ctxKeys.CLIENT_TYPE_KEY (set by AuthMiddleware from verified
// claims) — an absent value is treated as a browser, never inferred from unverified request data.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		token, err := ensureToken(wtr, req)
		if err != nil {
			logger.Error("failed to issue csrf token", err)
			httpErr.SendError(wtr, req, httpErr.Global.Internal, httpErr.WithDetail("failed to issue csrf token"))
			return
		}

		switch req.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if req.Context().Value(ctxKeys.CLIENT_TYPE_KEY) == "api" {
				ctx := context.WithValue(req.Context(), ctxKeys.CSRF_TOKEN_KEY, "api-client-exempt")
				ctx = context.WithValue(ctx, ctxKeys.CSRF_VALID_KEY, true)
				next.ServeHTTP(wtr, req.WithContext(ctx))
				return
			}

			if ok, err := headers.ValidateHeaderValue(headers.HEADER_X_CSRF_TOKEN, req); !ok || err != nil {
				logger.Error("invalid or missing CSRF token for browser client", err)
				httpErr.SendError(wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("invalid CSRF token"))
				return
			}
		}

		ctx := context.WithValue(req.Context(), ctxKeys.CSRF_TOKEN_KEY, token)
		ctx = context.WithValue(ctx, ctxKeys.CSRF_VALID_KEY, true)
		next.ServeHTTP(wtr, req.WithContext(ctx))
	})
}
