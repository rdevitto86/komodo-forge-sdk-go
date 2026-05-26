package middleware

import (
	mwcors "github.com/rdevitto86/komodo-forge-sdk-go/api/cors"
	mwcsrf "github.com/rdevitto86/komodo-forge-sdk-go/api/csrf"
	mwheaders "github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
	mwidempotency "github.com/rdevitto86/komodo-forge-sdk-go/api/idempotency"
	mwipaccess "github.com/rdevitto86/komodo-forge-sdk-go/api/ipaccess"
	mwchain "github.com/rdevitto86/komodo-forge-sdk-go/api/middleware/chain"
	mwnorm "github.com/rdevitto86/komodo-forge-sdk-go/api/normalization"
	mwratelimit "github.com/rdevitto86/komodo-forge-sdk-go/api/ratelimit"
	mwredaction "github.com/rdevitto86/komodo-forge-sdk-go/api/redaction"
	mwrequest "github.com/rdevitto86/komodo-forge-sdk-go/api/request"
	mwsanitize "github.com/rdevitto86/komodo-forge-sdk-go/api/sanitization"
	mwtelemetry "github.com/rdevitto86/komodo-forge-sdk-go/api/telemetry"
	mwapiauth "github.com/rdevitto86/komodo-forge-sdk-go/auth"
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
