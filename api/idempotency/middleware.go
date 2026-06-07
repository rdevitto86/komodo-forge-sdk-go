package idempotency

import (
	"context"
	"fmt"

	"net/http"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	"github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/api/request"
	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

var defaultStore *Store

func init() {
	defaultStore = NewStore("local", 0) // Initialize with local cache by default, can be configured via environment
}

// Replaces the default in-memory store with a custom one, typically a distributed cache.
func SetStore(store *Store) {
	defaultStore = store
}

// Guards against duplicate state-changing requests using the Idempotency-Key header; safe methods are skipped.
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

		// CheckAndSet is atomic — avoids the race a separate Check+Set allows
		allowed, err := defaultStore.CheckAndSet(key)
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

		next.ServeHTTP(wtr, req)
	})
}
