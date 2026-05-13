# komodo-forge-sdk-go

Internal Go SDK for all Komodo services. Provides AWS clients, HTTP middleware, JWT/OAuth crypto, structured logging, concurrency utilities, and a universal server entry point.

Module: `komodo-forge-sdk-go`

All services reference this SDK via a local `replace` directive in their `go.mod`:
```
replace komodo-forge-sdk-go => ../komodo-forge-sdk-go
```

---

## Packages

### `auth`
OAuth2 + JWT Bearer token authentication middleware.

```go
auth.AuthMiddleware(next http.Handler) http.Handler
```

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

### `aws/dynamo`
DynamoDB client with typed CRUD, query/scan helpers, and parallel batch operations.

```go
c, err := dynamo.New(config)
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

### `aws/elasticache`
Redis client via `go-redis`. Includes a distributed token-bucket rate limiter.

```go
c, err := elasticache.New(cfg)
c, err := elasticache.NewFromString(connStr)
c.Get(ctx, key)                                       // returns (string, error); "" on miss
c.Set(ctx, key, value, ttlSec)
c.Delete(ctx, key)
c.AllowDistributed(ctx, key, rate, burst, ttlSec)     // returns (allowed bool, retryAfter time.Duration, err)
c.Close()
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

### `aws/aurora`
Aurora-compatible SQL client (planned).

### `aws/bedrock`
Amazon Bedrock client (planned).

### `aws/cloudwatch`
CloudWatch client (planned).

### `aws/connect`
Amazon Connect client (planned).

### `aws/contactlens`
Contact Lens client (planned).

### `aws/elasticsearch`
Elasticsearch client (planned).

### `aws/lambda`
Lambda invocation client (planned).

### `aws/ses`
SES email client (planned).

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
c := client.NewClient(opts...)                        // options: WithTimeout, WithTransport, etc.
client.GetJSON[T](c, ctx, url)                        // returns (*T, error)
client.PostJSON[T](c, ctx, url, body)                 // returns (*T, error)
```

### `http/context`
Context key definitions and context enrichment middleware.

```go
httpctx.ContextMiddleware(next http.Handler) http.Handler
httpctx.GetUserID(ctx)
httpctx.GetRequestID(ctx)
// ... see http/context/getters.go
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

### `http/handlers`
Shared handler exports.

```go
handlers.HealthHandler  // http.HandlerFunc for /health
```

### `http/headers`
Header validation and evaluation middleware.

### `http/ipaccess`
IP whitelist/blacklist enforcement middleware.

### `http/middleware/chain`
Middleware chain composition.

```go
mwchain.Chain(handlerFunc, middleware...)  // returns http.Handler; first listed = outermost
```

### `http/normalization`
Request path and header normalization middleware.

### `http/ratelimit`
Token bucket rate limiting middleware. Reads `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`, and `ENV` from environment at startup.

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
request.RequestIDMiddleware(next http.Handler) http.Handler
request.ClientTypeMiddleware(next http.Handler) http.Handler
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
RS256 JWT signing and validation. Also importable at `crypto/jwt` (re-export shim).

```go
jwt.InitializeKeys()
jwt.SignToken(issuer, subject, audience, ttl, scopes)
jwt.ValidateToken(tokenString)
jwt.ParseClaims(tokenString)
jwt.ExtractTokenFromRequest(req)
```

### `security/oauth`
OAuth 2.0 scope and grant type validation. Also importable at `crypto/oauth` (re-export shim).

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
Universal service entry point. Auto-detects Lambda vs Fargate/local. Also importable at `http/server` (re-export shim).

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
