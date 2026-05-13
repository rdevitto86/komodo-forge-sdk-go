package sanitization

import (
	"bytes"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/http/errors"
)

// Sanitizes HTTP requests from malicious content
func SanitizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		sanitizeHeaders(req)
		sanitizePathParams(req)
		sanitizeQueryParams(req)

		if req.Body != nil && req.Header.Get("Content-Type") == "application/json" {
			sanitizeBody(wtr, req)
			if req.Body == nil {
				return
			}
		}

		next.ServeHTTP(wtr, req)
	})
}

// Removes malicious content from HTTP headers
func sanitizeHeaders(req *http.Request) {
	for key, values := range req.Header {
		for i, value := range values {
			req.Header[key][i] = sanitizeString(value)
		}
	}
}

// Sanitizes URL path parameters using stdlib routing (Go 1.22+)
func sanitizePathParams(req *http.Request) {
	pattern := req.Pattern
	if pattern == "" {
		return
	}

	// Extract wildcard names from the pattern (e.g. "GET /item/{sku}" -> ["sku"])
	for _, seg := range strings.Split(pattern, "/") {
		if len(seg) > 2 && seg[0] == '{' && seg[len(seg)-1] == '}' {
			name := seg[1 : len(seg)-1]
			// Strip trailing ... for catch-all wildcards
			name = strings.TrimSuffix(name, "...")
			if val := req.PathValue(name); val != "" {
				req.SetPathValue(name, sanitizeString(val))
			}
		}
	}
}

// Sanitizes URL query parameters
func sanitizeQueryParams(req *http.Request) {
	query := req.URL.Query()
	sanitized := url.Values{}

	for key, values := range query {
		sanitizedKey := sanitizeString(key)
		for _, value := range values {
			sanitized.Add(sanitizedKey, sanitizeString(value))
		}
	}

	req.URL.RawQuery = sanitized.Encode()
}

// Sanitizes JSON request body
func sanitizeBody(wtr http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		req.Body = nil
		httpErr.SendError(wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("failed to read request body"))
		return
	}
	req.Body.Close()

	// Parse JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		req.Body = nil
		httpErr.SendError(wtr, req, httpErr.Global.BadRequest, httpErr.WithDetail("failed to parse JSON body"))
		return
	}

	// Sanitize the data recursively
	sanitized := sanitizeJSON(data)

	// Re-encode to JSON — json.Marshal of a sanitized interface{} tree (containing
	// only strings, maps, arrays, and primitives) is always successful.
	sanitizedBody, _ := json.Marshal(sanitized)

	// Replace request body with sanitized version
	req.Body = io.NopCloser(bytes.NewBuffer(sanitizedBody))
	req.ContentLength = int64(len(sanitizedBody))
}

// Recursively sanitizes JSON data structures
func sanitizeJSON(data interface{}) interface{} {
	switch val := data.(type) {
	case string:
		return sanitizeString(val)
	case map[string]interface{}:
		sanitized := make(map[string]interface{})
		for key, value := range val {
			sanitizedKey := sanitizeString(key)
			sanitized[sanitizedKey] = sanitizeJSON(value)
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(val))
		for i, value := range val {
			sanitized[i] = sanitizeJSON(value)
		}
		return sanitized
	default:
		return val
	}
}

// Sanitizes strings from malicious patterns
func sanitizeString(str string) string {
	str = NullBytePattern.ReplaceAllString(str, "")
	str = PathTraversalPattern.ReplaceAllString(str, "")
	str = html.EscapeString(str)
	str = strings.TrimSpace(str)

	if SqlInjectionPattern.MatchString(str) {
		str = SqlInjectionPattern.ReplaceAllString(str, "")
	}
	str = XssPattern.ReplaceAllString(str, "")

	return str
}
