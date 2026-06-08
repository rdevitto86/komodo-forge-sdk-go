package request

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	proxyDepthLoadOnce sync.Once
	trustedProxyDepth  atomic.Int32
)

// Sets how many trusted reverse-proxy hops sit in front of the service, controlling how
// far from the right of X-Forwarded-For GetClientKey reads the client IP. A depth of 0
// (the default) ignores the header and uses the peer; overrides TRUSTED_PROXY_DEPTH.
func SetTrustedProxyDepth(n int) {
	// Mark the env loader as run so it cannot later overwrite this explicit setting.
	proxyDepthLoadOnce.Do(func() {})
	if n < 0 {
		n = 0
	}
	trustedProxyDepth.Store(int32(n))
}

// Returns the configured trusted-proxy depth, loading it once from TRUSTED_PROXY_DEPTH
// unless SetTrustedProxyDepth has already been called.
func clientTrustDepth() int {
	proxyDepthLoadOnce.Do(func() {
		if v := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_DEPTH")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				trustedProxyDepth.Store(int32(n))
			}
		}
	})
	return int(trustedProxyDepth.Load())
}

// Extracts the API version from Accept or Content-Type headers (primary) or the URL path prefix as a fallback.
func GetAPIVersion(req *http.Request) string {
	if req == nil {
		return ""
	}

	// Priority 1: Check Accept header for version parameter
	if accept := req.Header.Get("Accept"); accept != "" {
		if version := extractVersionFromMediaType(accept); version != "" {
			return version
		}
	}

	// Priority 2: Check Content-Type header for version parameter
	if contentType := req.Header.Get("Content-Type"); contentType != "" {
		if version := extractVersionFromMediaType(contentType); version != "" {
			return version
		}
	}

	// Priority 3: Fallback to URL path versioning (e.g., /v1/resource)
	if req.URL != nil {
		trimmed := strings.TrimPrefix(req.URL.Path, "/")
		segments := strings.Split(trimmed, "/")

		if len(segments) > 0 && len(segments[0]) > 0 && segments[0][0] == 'v' {
			return "/" + segments[0]
		}
	}
	return ""
}

// Extracts version from media type header (e.g., "application/json;v=1" or "application/json; version=2")
func extractVersionFromMediaType(mediaType string) string {
	parts := strings.Split(mediaType, ";")
	if len(parts) < 2 {
		return ""
	}

	for _, part := range parts[1:] {
		param := strings.TrimSpace(part)

		// Support both "v=1" and "version=1" formats
		if version, ok := strings.CutPrefix(param, "v="); ok {
			return "/v" + strings.TrimSpace(version)
		}
		if version, ok := strings.CutPrefix(param, "version="); ok {
			return "/v" + strings.TrimSpace(version)
		}
	}
	return ""
}

// Extracts the API route from the request URL, excluding version prefix if present.
func GetAPIRoute(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}

	var base string = req.URL.Path
	if idx := strings.Index(req.URL.Path, "?"); idx != -1 {
		base = req.URL.Path[:idx]
	}

	// Split path and detect version segment if present
	trimmed := strings.TrimPrefix(base, "/")
	segments := strings.Split(trimmed, "/")

	var pathSegments = []string{}

	if len(segments) > 0 && len(segments[0]) > 0 && segments[0][0] == 'v' {
		pathSegments = segments[1:]
	} else {
		pathSegments = segments // No explicit version prefix
	}

	// Route is the path without version
	route := "/" + strings.Join(pathSegments, "/")
	if route == "//" {
		route = "/"
	}
	return route
}

// Extracts path parameters from the request URL; placeholder — replace with actual routing-aware logic.
func GetPathParams(req *http.Request) map[string]string {
	// Placeholder: return empty map as path parameter extraction requires route pattern knowledge
	return map[string]string{}
}

// Extracts the first value of each query parameter from the request URL.
func GetQueryParams(req *http.Request) map[string]string {
	if req == nil || req.URL == nil {
		return map[string]string{}
	}

	out := make(map[string]string)
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return out
	}

	for k, v := range values {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

// Checks only the X-API-Key header — Bearer claims are unverified at this layer and forgeable; prefer
// ctxKeys.CLIENT_TYPE_KEY, which AuthMiddleware derives from verified claims.
func GetClientType(req *http.Request) string {
	if apiKey := req.Header.Get("X-API-Key"); apiKey != "" && IsValidAPIKey(apiKey) {
		return "api"
	}
	return "browser"
}

// Validates that an API key exists and is active; placeholder — replace with DynamoDB/RDS lookup when the database is ready.
func IsValidAPIKey(apiKey string) bool {
	// Placeholder: Replace with actual database lookup
	// Expected implementation:
	// 1. Query DynamoDB/RDS for api_key
	// 2. Check if key exists and is active (not revoked/expired)
	// 3. Optional: Rate limit check, scope validation
	// 4. Log the API key usage for auditing

	return true
}

// Extracts a client identifier from the request: the direct peer address by default,
// or the originating client from X-Forwarded-For when a trusted-proxy depth is configured.
// Never trusts the client-supplied leftmost hop, which would let callers spoof the key
// and bypass rate limiting and IP access control.
func GetClientKey(req *http.Request) string {
	if depth := clientTrustDepth(); depth > 0 {
		if xf := req.Header.Get("X-Forwarded-For"); xf != "" {
			parts := strings.Split(xf, ",")
			// With `depth` trusted proxies in front, the originating client is the entry
			// `depth` positions from the right (index len-depth); anything further left
			// may be a client-injected spoof and is ignored. A header shorter than depth
			// signals spoofing or misconfig — fall through to the unspoofable peer.
			if idx := len(parts) - depth; idx >= 0 && idx < len(parts) {
				if ip := strings.TrimSpace(parts[idx]); ip != "" {
					return ip
				}
			}
		}
	}
	// Fall back to the direct peer host — the only source a client cannot forge.
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return req.RemoteAddr
}
