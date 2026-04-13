# komodo-forge-sdk-go

Internal Go SDK for all Komodo services. Provides AWS clients, HTTP middleware, JWT/OAuth crypto, structured logging, concurrency utilities, and a universal server entry point.

Module: `komodo-forge-sdk-go`

All services reference this SDK via a local `replace` directive in their `go.mod`:
```
replace komodo-forge-sdk-go => ../komodo-forge-sdk-go
```

---

## Packages

### `api/auth`
Authentication utilities.

### `api/circuitbreaker`
Circuit breaker implementation (planned).

### `api/idempotency`
Idempotency key deduplication middleware.

### `api/rules`
YAML-driven field validation rules.

### `aws/aurora`
Aurora-compatible SQL client (planned).

### `aws/dynamo`
DynamoDB client with typed CRUD and query/scan helpers.

```go
dynamo.Init(cfg)
dynamo.GetItem(ctx, table, key, &out)
dynamo.WriteItem(ctx, table, item)
dynamo.UpdateItem(ctx, table, key, updates)
dynamo.DeleteItem(ctx, table, key)
dynamo.Query(ctx, input)
dynamo.QueryAll(ctx, input)
dynamo.Scan(ctx, input)
```

### `aws/elasticache`
Redis client via `go-redis`. Includes distributed token-bucket rate limiter.

```go
elasticache.Init(cfg)
elasticache.Get(key)
elasticache.Set(key, value, ttlSec)
elasticache.Delete(key)
elasticache.AllowDistributed(ctx, key, rate, burst, ttlSec) // returns (allowed bool, retryAfter time.Duration, err)
```

### `aws/s3`
S3 client with typed get/put/delete.

```go
awsS3.Init(cfg)
awsS3.GetObject(ctx, bucket, key)           // returns []byte
awsS3.GetObjectAs(ctx, bucket, key, &out)   // unmarshals JSON into out
awsS3.PutObject(ctx, bucket, key, data, contentType)
awsS3.DeleteObject(ctx, bucket, key)
```

### `aws/secretsmanager`
Bootstraps AWS Secrets Manager at service startup. Resolves secrets into in-memory config.

```go
awsSM.Bootstrap(cfg)
awsSM.GetSecret(key, prefix)
```

### `config`
In-memory config store. Checks local store first, falls back to `os.Getenv`. Used by the Secrets Manager bootstrap to inject resolved secrets.

```go
config.GetConfigValue(key)
config.SetConfigValue(key, val)
config.DeleteConfigValue(key)
config.GetAllKeys()
```

### `connectors`
Third-party service connectors:
- `connectors/stripe` — Stripe payment integration
- `connectors/paypal` — PayPal integration
- `connectors/apple` — Apple services
- `connectors/google` — Google services

### `events`
Event publishing and envelope handling (planned).

### `http/chain`
Middleware chain composition.

```go
chain.New(middleware...)
```

### `http/client`
HTTP client wrapper (planned - configurable timeouts, retry logic, middleware pipeline, observability).

```go
client.NewClient()
```

### `http/context`
Context key definitions and context enrichment middleware.

```go
context.ContextMiddleware(next)
```

### `http/cors`
CORS middleware (planned).

### `http/csrf`
CSRF token validation middleware.

### `http/errors`
RFC 7807 Problem+JSON error responses.

```go
errors.SendError(w, r, errCode, overrides...)
errors.SendCustomError(w, r, status, message, detail, code)
errors.WithMessage(msg)
errors.WithDetail(detail)
errors.WithStatus(status)
```

### `http/headers`
Header validation and evaluation.

### `http/ipaccess`
IP whitelist/blacklist enforcement middleware.

### `http/normalization`
Request path and header normalization middleware.

### `http/ratelimit`
Token bucket rate limiting middleware.

### `http/redaction`
Sensitive field redaction from logs.

### `http/request`
Request parsing helpers and middleware.

```go
request.GetPathParams(req)
request.GetQueryParams(req)
request.GetClientType(req)
request.GetRequestID(req)
request.GenerateRequestId()
request.RequestIDMiddleware(next)
request.ClientTypeMiddleware(next)
```

### `http/response`
Response writer wrapper with status tracking.

```go
response.Bind(res, &target)
response.IsSuccess(status)
response.IsError(status)
```

### `http/sanitization`
Input sanitization (XSS, injection) middleware.

### `http/telemetry`
Request/response telemetry logging middleware.

### `logging/otel`
OpenTelemetry integration (planned).

### `logging/runtime`
`slog`-based structured logger. `tint` handler for local/dev, JSON handler for staging/prod. Includes automatic field redaction.

```go
logger.Init(appName, logLevel, env)
logger.Debug(msg, args...)
logger.Info(msg, args...)
logger.Warn(msg, args...)
logger.Error(msg, args...)
logger.Fatal(msg, err)
```

### `security/encryption`
Encryption utilities (planned).

### `security/jwt`
RS256 JWT signing and validation.

```go
jwt.InitializeKeys()
jwt.SignToken(issuer, subject, audience, ttl, scopes)
jwt.ValidateToken(tokenString)
jwt.ParseClaims(tokenString)
jwt.ExtractTokenFromRequest(req)
```

### `security/oauth`
OAuth 2.0 scope and grant type validation.

```go
oauth.IsValidScope(scope)
oauth.IsValidGrantType(grantType)
```

### `server`
Universal service entry point. Auto-detects Lambda vs Fargate/local.

```go
// On AWS Lambda (AWS_LAMBDA_FUNCTION_NAME set): starts lambda.Start with httpadapter
// Otherwise: ListenAndServe with graceful SIGINT/SIGTERM shutdown
server.Run(srv, port, gracefulTimeout)
```

---

## Testing Utilities

### `testing/chaos`
Chaos injection helpers — latency simulation and error injection.

### `testing/moxtox`
HTTP mock server builder for unit testing middleware and handlers without network calls.

### `testing/performance`
Latency measurement and benchmarking utilities.

---

## Development

```bash
./scripts/test.sh     # run all tests with race detector
./scripts/lint.sh     # run golangci-lint
./scripts/audit.sh    # security audit
./scripts/coverage.sh # generate coverage report
```
