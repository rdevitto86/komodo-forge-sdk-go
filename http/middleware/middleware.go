// Package middleware re-exports the full public middleware stack as a single import point.
// Services import this package as: mw "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware"
package middleware

import (
	"net/http"

	mwapiauth "github.com/rdevitto86/komodo-forge-sdk-go/api/auth"
	mwidempotency "github.com/rdevitto86/komodo-forge-sdk-go/api/idempotency"
	mwrules "github.com/rdevitto86/komodo-forge-sdk-go/api/rules"
	mwcors "github.com/rdevitto86/komodo-forge-sdk-go/http/cors"
	mwcsrf "github.com/rdevitto86/komodo-forge-sdk-go/http/csrf"
	mwheaders "github.com/rdevitto86/komodo-forge-sdk-go/http/headers"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/mwchain"
	mwnorm "github.com/rdevitto86/komodo-forge-sdk-go/http/normalization"
	mwratelimit "github.com/rdevitto86/komodo-forge-sdk-go/http/ratelimit"
	mwrequest "github.com/rdevitto86/komodo-forge-sdk-go/http/request"
	mwsanitize "github.com/rdevitto86/komodo-forge-sdk-go/http/sanitization"
	mwipaccess "github.com/rdevitto86/komodo-forge-sdk-go/http/ipaccess"
	mwtelemetry "github.com/rdevitto86/komodo-forge-sdk-go/http/telemetry"
)

// Chain applies middleware in order: first listed = outermost wrapper.
func Chain(f func(http.ResponseWriter, *http.Request), mw ...func(http.Handler) http.Handler) http.Handler {
	return mwchain.Chain(f, mw...)
}

// RequestIDMiddleware attaches a request ID to the context and response headers.
var RequestIDMiddleware = mwrequest.RequestIDMiddleware

// ClientSourceMiddleware extracts and attaches client source info to the context.
var ClientSourceMiddleware = mwrequest.ClientSourceMiddleware

// ClientTypeMiddleware is an alias for ClientSourceMiddleware retained for backward compatibility.
var ClientTypeMiddleware = mwrequest.ClientSourceMiddleware

// TelemetryMiddleware records request telemetry (latency, status, route).
var TelemetryMiddleware = mwtelemetry.TelemetryMiddleware

// RateLimiterMiddleware enforces per-client rate limits.
var RateLimiterMiddleware = mwratelimit.RateLimiterMiddleware

// AuthMiddleware validates the JWT in the Authorization header and attaches claims to context.
var AuthMiddleware = mwapiauth.AuthMiddleware

// RequireServiceScope validates that the JWT carries the required service-to-service scope.
var RequireServiceScope = mwapiauth.RequireServiceScope

// ScopeMiddleware is an alias for RequireServiceScope retained for backward compatibility.
var ScopeMiddleware = mwapiauth.RequireServiceScope

// CORSMiddleware sets CORS response headers.
var CORSMiddleware = mwcors.CORSMiddleware

// SecurityHeadersMiddleware sets security-related response headers (HSTS, CSP, etc.).
var SecurityHeadersMiddleware = mwheaders.SecurityHeadersMiddleware

// CSRFMiddleware validates CSRF tokens on mutating requests.
var CSRFMiddleware = mwcsrf.CSRFMiddleware

// NormalizationMiddleware normalizes headers, URL paths, and query params.
var NormalizationMiddleware = mwnorm.NormalizationMiddleware

// SanitizationMiddleware sanitizes request headers, path params, query params, and body.
var SanitizationMiddleware = mwsanitize.SanitizationMiddleware

// IdempotencyMiddleware enforces idempotency on mutating requests via idempotency keys.
var IdempotencyMiddleware = mwidempotency.IdempotencyMiddleware

// RuleValidationMiddleware validates incoming requests against registered rule sets.
var RuleValidationMiddleware = mwrules.RuleValidationMiddleware

// IPAccessMiddleware enforces IP allowlist/blocklist rules.
var IPAccessMiddleware = mwipaccess.IPAccessMiddleware
