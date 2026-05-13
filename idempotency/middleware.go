package idempotency

import (
	"context"
	"fmt"

	"net/http"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/http/errors"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/headers"
	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/http/request"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

var defaultStore *Store

func init() {
	defaultStore = NewStore("local", 0) // Initialize with local cache by default, can be configured via environment
}

// Sets a custom store for idempotency (useful for distributed cache)
func SetStore(store *Store) {
	defaultStore = store
}

// Guards against duplicate requests using the Idempotency-Key header.
// It only applies to unsafe, state-changing methods (POST, PUT, PATCH, DELETE).
// Safe methods (GET, HEAD, OPTIONS) are skipped entirely.
func IdempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		// Only guard unsafe, state-changing methods
		switch req.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(wtr, req)
			return
		}

		clientType := req.Context().Value(ctxKeys.CLIENT_TYPE_KEY)
		if clientType == nil {
			clientType = httpReq.GetClientType(req)
		}

		if clientType == "api" {
			ctx := context.WithValue(req.Context(), ctxKeys.IDEMPOTENCY_VALID_KEY, true)
			req = req.WithContext(ctx)
			next.ServeHTTP(wtr, req)
			return
		}

		key := req.Header.Get("Idempotency-Key")

		if ok, err := headers.ValidateHeaderValue(headers.HEADER_IDEMPOTENCY, req); !ok || err != nil {
			logger.Error("invalid idempotency key for browser client: "+key, err)
			httpErr.SendError(
				wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("invalid idempotency key"),
			)
			return
		}

		// Check if key already exists
		allowed, err := defaultStore.Check(key)
		if err != nil {
			logger.Error("idempotency check failed for key: "+key, err)
			httpErr.SendError(
				wtr, req, httpErr.Global.Internal, httpErr.WithDetail("idempotency check failed"),
			)
			return
		}

		if !allowed {
			wtr.Header().Set("Idempotency-Replayed", "true")
			logger.Error("duplicate request: "+key, fmt.Errorf("duplicate request"))
			httpErr.SendError(
				wtr, req, httpErr.Global.Conflict, httpErr.WithDetail("duplicate request"),
			)
			return
		}

		req = req.WithContext(context.WithValue(
			req.Context(), ctxKeys.IDEMPOTENCY_VALID_KEY, true,
		))

		// Store key with expiration
		if err := defaultStore.Set(key); err != nil {
			logger.Error("failed to store idempotency key: "+key, err)
			// Continue anyway - the request is allowed, just logging the failure
		}

		next.ServeHTTP(wtr, req)
	})
}
