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
	"github.com/rdevitto86/komodo-forge-sdk-go/auth"
	mwvalidation "github.com/rdevitto86/komodo-forge-sdk-go/api/validation"
)

var (
	Chain = mwchain.Chain

	AuthMiddleware             = auth.AuthMiddleware
	Middleware                 = auth.Middleware
	ClientSourceMiddleware     = mwrequest.ClientSourceMiddleware
	ClientTypeMiddleware       = mwrequest.ClientSourceMiddleware
	CORSMiddleware             = mwcors.CORSMiddleware
	CSRFMiddleware             = mwcsrf.CSRFMiddleware
	IdempotencyMiddleware      = mwidempotency.IdempotencyMiddleware
	IPAccessMiddleware         = mwipaccess.IPAccessMiddleware
	MaxContentLengthMiddleware = mwheaders.MaxContentLengthMiddleware
	NormalizationMiddleware    = mwnorm.NormalizationMiddleware
	RateLimiterMiddleware      = mwratelimit.RateLimiterMiddleware
	RedactionMiddleware        = mwredaction.RedactionMiddleware
	RequestIDMiddleware        = mwrequest.RequestIDMiddleware
	RequireAnyScope            = auth.RequireAnyScope
	RequireServiceScope        = auth.RequireServiceScope
	RuleValidationMiddleware   = mwvalidation.RuleValidationMiddleware
	ValidationMiddleware       = mwvalidation.Middleware
	SanitizationMiddleware     = mwsanitize.SanitizationMiddleware
	ScopeMiddleware            = auth.RequireServiceScope
	SecurityHeadersMiddleware  = mwheaders.SecurityHeadersMiddleware
	TelemetryMiddleware        = mwtelemetry.TelemetryMiddleware
)
