package request

import (
	"context"
	"net/http"

	httpCtx "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

// Ensures each request carries a unique X-Request-ID and propagates X-Correlation-ID into context and response headers.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var reqID string
		if rid := req.Header.Get("X-Request-ID"); rid != "" {
			reqID = rid
		} else if rid, ok := req.Context().Value(httpCtx.REQUEST_ID_KEY).(string); ok && rid != "" {
			reqID = rid
		} else {
			reqID = GenerateRequestId()
		}

		req.Header.Set("X-Request-ID", reqID)
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(req.Context(), httpCtx.REQUEST_ID_KEY, reqID)

		// Correlation ID (browser fingerprint — client-generated, server echoes back)
		if cid := req.Header.Get("X-Correlation-ID"); cid != "" {
			ctx = context.WithValue(ctx, httpCtx.CORRELATION_ID_KEY, cid)
			w.Header().Set("X-Correlation-ID", cid)
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

const (
	ClientTypeAPI     string = "api"
	ClientTypeBrowser string = "browser"
)

// Detects whether the request originates from an API client or browser and stores the result in context.
func ClientSourceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		hasReferer := req.Header.Get("Referer") != ""
		hasCookie := req.Header.Get("Cookie") != ""

		clientType := ClientTypeBrowser
		if authHeader != "" && !hasReferer && !hasCookie {
			clientType = ClientTypeAPI
		}

		ctx := context.WithValue(req.Context(), httpCtx.CLIENT_TYPE_KEY, clientType)
		next.ServeHTTP(wtr, req.WithContext(ctx))
	})
}
