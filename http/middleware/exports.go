// Package middleware re-exports the full public middleware stack as a single import point.
package middleware

import (
	mwapiauth "github.com/rdevitto86/komodo-forge-sdk-go/auth"
	mwcors "github.com/rdevitto86/komodo-forge-sdk-go/http/cors"
	mwcsrf "github.com/rdevitto86/komodo-forge-sdk-go/http/csrf"
	mwheaders "github.com/rdevitto86/komodo-forge-sdk-go/http/headers"
	mwipaccess "github.com/rdevitto86/komodo-forge-sdk-go/http/ipaccess"
	mwchain "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/chain"
	mwnorm "github.com/rdevitto86/komodo-forge-sdk-go/http/normalization"
	mwratelimit "github.com/rdevitto86/komodo-forge-sdk-go/http/ratelimit"
	mwredaction "github.com/rdevitto86/komodo-forge-sdk-go/http/redaction"
	mwrequest "github.com/rdevitto86/komodo-forge-sdk-go/http/request"
	mwsanitize "github.com/rdevitto86/komodo-forge-sdk-go/http/sanitization"
	mwtelemetry "github.com/rdevitto86/komodo-forge-sdk-go/http/telemetry"
	mwidempotency "github.com/rdevitto86/komodo-forge-sdk-go/idempotency"
	mwrules "github.com/rdevitto86/komodo-forge-sdk-go/rules"
)

var (
	// --- core ---

	Chain = mwchain.Chain

	// --- handlers ---

	AuthMiddleware            = mwapiauth.AuthMiddleware
	ClientSourceMiddleware    = mwrequest.ClientSourceMiddleware
	ClientTypeMiddleware      = mwrequest.ClientSourceMiddleware
	CORSMiddleware            = mwcors.CORSMiddleware
	CSRFMiddleware            = mwcsrf.CSRFMiddleware
	IdempotencyMiddleware     = mwidempotency.IdempotencyMiddleware
	IPAccessMiddleware        = mwipaccess.IPAccessMiddleware
	NormalizationMiddleware   = mwnorm.NormalizationMiddleware
	RateLimiterMiddleware     = mwratelimit.RateLimiterMiddleware
	RedactionMiddleware       = mwredaction.RedactionMiddleware
	RequestIDMiddleware       = mwrequest.RequestIDMiddleware
	RequireServiceScope       = mwapiauth.RequireServiceScope
	RuleValidationMiddleware  = mwrules.RuleValidationMiddleware
	SanitizationMiddleware    = mwsanitize.SanitizationMiddleware
	ScopeMiddleware           = mwapiauth.RequireServiceScope
	SecurityHeadersMiddleware = mwheaders.SecurityHeadersMiddleware
	TelemetryMiddleware       = mwtelemetry.TelemetryMiddleware
)
