package middleware

import (
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/auth"
	clienttype "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/client-type"

	// "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/context"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/chain"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/cors"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/csrf"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/idempotency"
	ipaccess "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/ip-access"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/normalization"
	ratelimiter "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/rate-limiter"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/redaction"
	requestid "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/request-id"
	rulevalidation "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/rule-validation"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/sanitization"
	"github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/scope"
	securityheaders "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/security-headers"
	telemetry "github.com/rdevitto86/komodo-forge-sdk-go/http/middleware/telemetry"
)

var (
	AuthMiddleware = auth.AuthMiddleware
	Chain = chain.Chain
	ClientTypeMiddleware = clienttype.ClientTypeMiddleware
	// ContextMiddleware = context.ContextMiddleware
	CORSMiddleware = cors.CORSMiddleware
	CSRFMiddleware = csrf.CSRFMiddleware
	IdempotencyMiddleware = idempotency.IdempotencyMiddleware
	IPAccessMiddleware = ipaccess.IPAccessMiddleware
	NormalizationMiddleware = normalization.NormalizationMiddleware
	RateLimiterMiddleware = ratelimiter.RateLimiterMiddleware
	RedactionMiddleware = redaction.RedactionMiddleware
	RequestIDMiddleware = requestid.RequestIDMiddleware
	RuleValidationMiddleware = rulevalidation.RuleValidationMiddleware
	SanitizationMiddleware = sanitization.SanitizationMiddleware
	SecurityHeadersMiddleware = securityheaders.SecurityHeadersMiddleware
	ScopeMiddleware = scope.RequireServiceScope
	TelemetryMiddleware = telemetry.TelemetryMiddleware
)
