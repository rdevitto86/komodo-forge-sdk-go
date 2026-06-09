# komodo-forge-sdk-go

Internal Go SDK for all Komodo services. Provides AWS clients, HTTP middleware, JWT/OAuth crypto, structured logging, concurrency utilities, and a universal server entry point.

Module: `github.com/rdevitto86/komodo-forge-sdk-go`

Services consume this SDK as a versioned Git dependency — no local `replace` directive:
```
go get github.com/rdevitto86/komodo-forge-sdk-go@vX.Y.Z
```

---

## Packages

### `auth`
JWT Bearer token verification and HTTP middleware. **This is the canonical token-verification path for every service.** Tokens are issued centrally by the Auth API (the sole holder of the private signing key); services verify-only via `JWKSVerifier` (RS256 public keys fetched from the JWKS endpoint) and never mint their own tokens.

```go
// Construct a verifier backed by the auth-api JWKS endpoint.
// ExpectedAudience and ExpectedIssuer are required — tokens for another audience are rejected.
v, err := auth.NewJWKSVerifier(auth.Config{
    JWKSURL:          "https://auth.internal/.well-known/jwks.json",
    ExpectedAudience: "order-api",          // this service's identity
    ExpectedIssuer:   "https://auth.internal",
    CacheTTL:         5 * time.Minute,      // default; keys cached by kid
})

// Use the injected-Verifier middleware (preferred — testable).
router.Use(auth.Middleware(v))

// Restrict an internal route to service-to-service callers (svc:<name> scope).
router.Use(auth.RequireServiceScope)

// Sentinel errors for branching on failure mode.
claims, err := v.Verify(ctx, tokenString)
switch {
case errors.Is(err, auth.ErrExpired):          // token past exp
case errors.Is(err, auth.ErrInvalidSignature): // key mismatch / tamper
case errors.Is(err, auth.ErrInvalidToken):     // malformed / unknown kid
}

// Deprecated: use Middleware(v) instead.
auth.AuthMiddleware(next http.Handler) http.Handler
```

To **call** another internal service, obtain a service token from the Auth API with the
`client_credentials` grant and attach it automatically — no private key required (see
`http/client.WithServiceAuth` below).

### `config`
Well-known environment variable key constants shared across all services.

```go
config.APP_NAME      // "APP_NAME"
config.ENV           // "ENV"
config.PORT          // "PORT"
config.AWS_REGION    // "AWS_REGION"
config.JWT_PUBLIC_KEY // "JWT_PUBLIC_KEY"
// ... see config/config.go for the full list
```

### `idempotency`
Idempotency key deduplication with pluggable cache backend.

```go
idempotency.SetStore(store)          // set cache backend (LocalCache or distributed)
idempotency.IdempotencyMiddleware(next http.Handler) http.Handler
```

### `rules`
YAML-driven request field validation. Regex patterns are compiled once at load time.

```go
rules.RuleValidationMiddleware(next http.Handler) http.Handler
```

---

## AWS Packages

> **`aws/X`** wraps `aws-sdk-go-v2/service/X`. SigV4 auth, AWS-specific endpoints, SDK-managed transport. May cover control plane (provisioning) and/or AWS-proprietary data planes (DynamoDB, S3, RDS Data API).
>
> **`db/X`** wraps a protocol-native client (`pgx`, `go-redis`, `opensearch-go`). Caller-managed connection pool, explicit `Ping`/`Close`, portable across deployments (AWS, GCP, self-hosted, local).
>
> The same logical service can appear in both trees: e.g., `aws/elasticache` (cluster management via SDK) + `db/redis` (RESP data plane). The import path tells you which surface you're using.

### `aws/dynamodb`
DynamoDB client with typed CRUD, query/scan helpers, and parallel batch operations.

```go
c, err := dynamodb.New(config)
c.BuildKey(pk, pv, sk, sv)
c.GetItem(ctx, table, key, batch, keys)
c.GetItemAs(ctx, table, key, batch, keys, &out)
c.WriteItem(ctx, table, item, batch, items, condition)
c.WriteItemFrom(ctx, table, item, batch, items, condition)
c.UpdateItem(ctx, table, key, expr, exprVals, exprNames, condition)
c.UpdateItemAs(ctx, table, key, expr, exprVals, exprNames, condition, &out)
c.DeleteItem(ctx, table, key, batch, keys, condition)
c.Query(ctx, input)
c.QueryAs(ctx, input, &out)
c.QueryAll(ctx, input)
c.QueryAllAs(ctx, input, &out)
c.Scan(ctx, input)
c.ScanAs(ctx, input, &out)
c.ScanAll(ctx, input)
c.ScanAllAs(ctx, input, &out)
```

### `aws/s3`
S3 client with typed get/put/delete.

```go
c, err := s3.New(config)
c.GetObject(ctx, bucket, key)                         // returns []byte
c.GetObjectAs(ctx, bucket, key, &out)                 // unmarshals JSON into out
c.PutObject(ctx, bucket, key, data, contentType)
c.DeleteObject(ctx, bucket, key)
```

### `aws/secretsmanager`
AWS Secrets Manager client with in-process TTL cache (default 5 min).

```go
c, err := secretsmanager.New(cfg)
c.GetSecret(key, prefix)                              // single secret
c.GetSecrets(keys, prefix, batchID)                   // batch JSON blob, returns map[string]string
```

### `aws/sns`
SNS client supporting standard and FIFO topics.

```go
c, err := sns.New(config)
c.Publish(ctx, sns.PublishInput{TopicARN, Message, GroupID, DedupID, Attrs})
```

### `aws/sqs`
SQS client supporting standard and FIFO queues.

```go
c, err := sqs.New(config)
c.Send(ctx, sqs.SendInput{QueueURL, Body, GroupID, DedupID, Attrs})  // returns messageID
c.Receive(ctx, queueURL, maxMessages)                                 // returns []Message
c.Delete(ctx, queueURL, receiptHandle)
```

### `aws/bedrock`
Amazon Bedrock generative-AI inference. Typed `Model` enum (Claude Opus 4.7 / Sonnet 4.6 / Haiku 4.5, Titan Text, Llama 3, Mistral Large) with `ParseModel(string)` validation for HTTP handlers — invalid model names return `ErrUnknownModel` before any SDK call.

```go
c, err := bedrock.New(cfg)
m, err := bedrock.ParseModel(req.Model) // reject unknown models here
text, err := c.Invoke(ctx, m, "Hello")  // Anthropic-format wrapper
raw, err  := c.InvokeJSON(ctx, m, body) // raw passthrough
out, err  := c.Converse(ctx, input)     // model-agnostic chat
```

### `aws/cloudwatch/metrics`
CloudWatch metrics. Auto-chunks `PutMetricData` at 1000 datums per call.

```go
c, err := metrics.New(cfg)
c.PutMetricData(ctx, "Komodo/Test", []metrics.MetricDatum{...})
stats, err := c.GetMetricStatistics(ctx, in)
```

### `aws/cloudwatch/logs`
CloudWatch Logs. Auto-chunks `PutLogEvents` at 10000 events or ~1MB.

```go
c, err := logs.New(cfg)
c.PutLogEvents(ctx, group, stream, events)
out, err := c.FilterLogEvents(ctx, in)
```

### `aws/connect`
Amazon Connect voice-contact orchestration.

```go
c, err := connect.New(cfg)
id, err := c.StartOutboundVoiceContact(ctx, in)
attrs, err := c.GetContactAttributes(ctx, instanceID, contactID)
c.UpdateContactAttributes(ctx, instanceID, contactID, attrs)
flows, err := c.ListContactFlows(ctx, instanceID)
```

### `aws/connect/contactlens`
Contact Lens real-time call analytics. Sub-feature of Connect; nested structurally.

```go
c, err := contactlens.New(cfg)
segs, err := c.ListRealtimeContactAnalysisSegments(ctx, instanceID, contactID)
```

### `aws/elasticache`
ElastiCache cluster management via the AWS SDK (control plane). Data plane lives in `db/redis`.

```go
c, err := elasticache.New(cfg)
groups, err := c.DescribeReplicationGroups(ctx)
clusters, err := c.DescribeCacheClusters(ctx)
```

### `aws/lambda`
Lambda invocation.

```go
c, err := lambda.New(cfg)
payload, err := c.Invoke(ctx, fnName, body)  // sync
c.InvokeAsync(ctx, fnName, body)             // fire-and-forget
```

### `aws/ses`
SESv2 transactional email. Attachments encoded as `multipart/mixed` MIME.

```go
c, err := ses.New(cfg)
id, err := c.SendEmail(ctx, ses.SendEmailInput{
    From: "no-reply@x.com", To: []string{"u@y.com"},
    Subject: "Hi", TextBody: "...", HTMLBody: "<p>...</p>",
    Attachments: []ses.Attachment{{Filename: "r.pdf", ContentType: "application/pdf", Data: b}},
})
```

### `aws/rds`
RDS Data API for Aurora Serverless SQL-over-HTTPS. Distinct from `db/sql` (wire-protocol pgx). Use this for Lambda or out-of-VPC callers.

```go
c, err := rds.New(cfg) // ResourceArn + SecretArn required
out, err := c.ExecuteStatement(ctx, rds.ExecuteStatementInput{SQL: "SELECT $1", Parameters: map[string]any{"1": 42}})
tx, err  := c.BeginTransaction(ctx)
c.CommitTransaction(ctx, tx)
c.RollbackTransaction(ctx, tx)
```

### `aws/opensearch`
OpenSearch Service domain management via the AWS SDK (control plane). Data plane lives in `db/opensearch`.

```go
c, err := opensearch.New(cfg)
d, err := c.DescribeDomain(ctx, name)
names, err := c.ListDomainNames(ctx)
```

---

## DB Packages

### `db/sql`
Driver-agnostic SQL client (planned). Wraps `database/sql` with a `pgx` or MySQL driver.

### `db/redis`
Redis data-plane client via `go-redis`. Includes a distributed token-bucket rate limiter.

```go
c, err := redis.New(cfg)
c, err := redis.NewFromString(connStr)
c.Get(ctx, key)                                       // returns (string, error); "" on miss
c.Set(ctx, key, value, ttlSec)
c.Delete(ctx, key)
c.AllowDistributed(ctx, key, rate, burst, ttlSec)     // returns (allowed bool, retryAfter time.Duration, err)
c.Close()
```

### `db/opensearch`
OpenSearch data-plane client (planned). Wraps `opensearch-go`.

---

## Events

### `events`
Event publishing and consuming over SNS FIFO + SQS FIFO.

```go
// Publishing
p := events.NewPublisher(snsClient, topicARN)
p.Publish(ctx, event)

// Subscribing (blocking long-poll loop)
s := events.NewSubscriber(sqsClient, events.SubscriberConfig{QueueURL, MaxBatch})
s.Subscribe(ctx, func(ctx context.Context, event events.Event) error { ... })
```

---

## HTTP Packages

### `http/client`
HTTP client with circuit breaker, tuned transport, and typed JSON helpers.

```go
c := client.NewClient(opts...)                        // options: WithCircuitBreaker, WithTransport
client.GetJSON[T](c, ctx, url)                        // returns (*T, error)
client.PostJSON[T](c, ctx, url, body)                 // returns (*T, error)
```

**Service-to-service auth** (`WithServiceAuth`) — obtain a short-lived service token from the
Auth API via the OAuth2 `client_credentials` grant and attach it to every outbound request. The
token is cached and proactively refreshed; no private signing key is involved.

```go
src, err := client.NewClientCredentialsTokenSource(client.ServiceAuthConfig{
    TokenURL:     "https://auth.internal/v1/oauth/token",
    ClientID:     os.Getenv("SERVICE_CLIENT_ID"),
    ClientSecret: os.Getenv("SERVICE_CLIENT_SECRET"),
    Scope:        "svc:order-api",
})

// Compose into a Client (or any generated client's transport).
c := client.NewClient(client.ClientConfig{
    Transport: client.WithServiceAuth(nil, src),       // nil base = DefaultTransport
})
```

### `http/context`
Context key definitions and context enrichment middleware.

```go
httpctx.ContextMiddleware(next http.Handler) http.Handler
httpctx.GetUserID(ctx)
httpctx.GetRequestID(ctx)
// ... see http/context/getters.go
```

### `api/cors`
CORS middleware (planned).

### `api/csrf`
CSRF token validation middleware.

### `api/errors`
RFC 7807 Problem+JSON error responses.

```go
errors.SendError(w, r, errCode, overrides...)
errors.SendCustomError(w, r, status, message, detail, code)
errors.WithMessage(msg)
errors.WithDetail(detail)
errors.WithStatus(status)
```

### `api/handlers`
Shared handler exports.

```go
handlers.HealthHandler  // http.HandlerFunc for /health
```

### `api/headers`
Header validation and evaluation middleware.

### `api/ipaccess`
IP whitelist/blacklist enforcement middleware.

### `api/middleware/chain`
Middleware chain composition.

```go
mwchain.Chain(handlerFunc, middleware...)  // returns http.Handler; first listed = outermost
```

### `api/normalization`
Request path and header normalization middleware.

### `api/ratelimit`
Token bucket rate limiting middleware. Reads `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`, and `ENV` from environment at startup.

### `api/redaction`
Sensitive field redaction from logs.

### `api/request`
Request parsing helpers and middleware.

```go
request.GetPathParams(req)
request.GetQueryParams(req)
request.GetClientType(req)
request.GetRequestID(req)
request.GenerateRequestId()
request.RequestIDMiddleware(next http.Handler) http.Handler
request.ClientTypeMiddleware(next http.Handler) http.Handler
```

### `api/response`
Response writer wrapper with status tracking.

```go
response.Bind(res, &target)
response.IsSuccess(status)
response.IsError(status)
```

### `api/sanitization`
Input sanitization (XSS, injection) middleware.

### `api/telemetry`
Request/response telemetry logging middleware.

### `http/websocket`
WebSocket client wrapper via `gorilla/websocket`.

```go
c := websocket.NewClient(conn)
c.Write(messageType, data)
c.Read()    // returns (messageType int, data []byte, err error)
c.Close()
websocket.RouteHandler(w, r)  // upgrades the connection
```

---

## Crypto / Security

### `security/jwt`
RS256 JWT issuance and validation.

> **Issuance is Auth-API-only.** `InitializeKeys` (which loads the private signing key) and
> `SignToken` are intended for the central Auth API — the single service that holds the private
> key and runs the OAuth token endpoint. **Application services must not mint tokens:** verify
> inbound tokens via the [`auth`](#auth) package and obtain service tokens via
> [`http/client.WithServiceAuth`](#httpclient). The `crypto/jwt` re-export shim is **deprecated**.

```go
// Auth API only:
jwt.InitializeKeys()
jwt.SignToken(issuer, subject, audience, ttl, scopes)

// Verification helpers (static-key path; services should prefer auth.JWKSVerifier):
jwt.ValidateToken(tokenString)
jwt.ParseClaims(tokenString)
jwt.ExtractTokenFromRequest(req)
```

### `security/oauth`
OAuth 2.0 scope and grant type validation. The `crypto/oauth` re-export shim is **deprecated**.

```go
oauth.IsValidScope(scope)
oauth.GetInvalidScopes(scope)
oauth.IsValidGrantType(grantType)
```

### `security/encryption`
Encryption utilities (planned).

---

## Logging

### `logging/runtime`
`slog`-based structured logger. `tint` handler for local/dev, JSON handler for staging/prod. Includes automatic field redaction.

```go
logger.Init(appName, logLevel, env)
logger.Debug(msg, args...)
logger.Info(msg, args...)
logger.Warn(msg, args...)
logger.Error(msg, err)
logger.Fatal(msg, err)
```

### `logging/otel`
OpenTelemetry integration (planned).

---

## Connectors

Third-party service connectors:
- `connectors/stripe` — checkout, customer, payment intents, Apple Pay, Google Pay, refunds
- `connectors/paypal` — orders, payment sources, refunds
- `connectors/apple` — authentication, Maps
- `connectors/google` — authentication, Maps

---

## Server

### `server`
Universal service entry point. Auto-detects Lambda vs Fargate/local. Also importable at `api/server` (re-export shim).

```go
// On AWS Lambda (AWS_LAMBDA_FUNCTION_NAME set): starts lambda.Start with API Gateway v2 httpadapter.
// Otherwise: ListenAndServe with graceful SIGINT/SIGTERM shutdown.
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
