package request

import (
	"context"
	"net/http"

	httpCtx "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

// RequestIDMiddleware ensures each request has a unique X-Request-ID in both
// header and context. Priority: header (client-supplied) > context > generated.
//
// Also propagates X-Correlation-ID from the client — this is the browser's
// session fingerprint (correlationId) used to tie frontend and backend logs
// to the same browsing session.
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

// Detects whether request is from API client or browser
// and stores the result in context for downstream middleware to use
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
