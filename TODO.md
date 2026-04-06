# TODO

A running list of gaps, incomplete work, and planned additions. Each item is labeled **H** (high), **M** (medium), or **L** (low) priority and ordered within each section accordingly.

---

## In-Progress / Stubs to Complete

### `aws/aurora`
- [ ] **H** Implement Aurora RDS client (`client.go`, `errors.go` are empty stubs)
- [ ] **H** Connection pool configuration (max open/idle conns, lifetime)
- [ ] **H** Transaction support
- [ ] **M** Query builder helpers (select, insert, update, delete)
- [ ] **M** AWS Aurora-specific error wrapping

### `aws/dynamodb`
- [ ] **H** Retry logic for unprocessed items in batch write/delete
- [ ] **H** Transaction support (`TransactWriteItems`, `TransactGetItems`)
- [ ] **M** Conditional expression helpers beyond raw strings
- [ ] **L** Consistent read flag on Scan
- [ ] **L** Projection expression support

### `aws/elasticache`
- [ ] **H** Configurable timeouts (currently hardcoded to 3s/2s)
- [ ] **M** Connection pool configuration
- [ ] **M** Bulk ops: `MGET`, `MSET`, `MDEL`
- [ ] **L** Pub/sub support
- [ ] **L** Redis Cluster mode support

### `aws/s3`
- [ ] **H** AWS-specific error unwrapping (currently wraps all errors generically)
- [ ] **H** Streaming get (avoid loading full object into memory)
- [ ] **H** Pre-signed URL generation (get + put)
- [ ] **M** `HeadObject` and `ListObjects` / `ListObjectsV2`
- [ ] **M** Multipart upload (required for objects >5GB)
- [ ] **L** Bucket operations: create, delete, list

### `aws/secrets-manager`
- [ ] **H** Replace `context.TODO()` with proper timeout context in `GetSecret`/`GetSecrets`
- [ ] **H** Distinguish "not found" from other errors
- [ ] **M** Pagination for batch secret retrieval
- [ ] **L** Support binary secrets

### `concurrency/semaphore`
- [ ] **H** Implement semaphore primitive (file is an empty stub)

### `concurrency/worker`
- [ ] **H** Per-job timeout (context-independent)
- [ ] **M** Metrics: completed jobs, failed jobs, queue depth
- [ ] **M** Worker health checks
- [ ] **L** Job priority queue

### `config`
- [ ] **H** File-based config loading (YAML / JSON)
- [ ] **M** Multi-environment support (dev / staging / prod profiles)
- [ ] **M** Thread-safe `SetLevel` for log level changes
- [ ] **L** Change notification / listener hooks

### `crypto/jwt`
- [ ] **H** Token revocation / JTI blacklist (revocation check is currently commented out)
- [ ] **H** Token refresh / rotation mechanism
- [ ] **M** Support for multiple concurrent key versions (JWKS-style)
- [ ] **M** Token introspection
- [ ] **L** Key pair generation helper (currently assumes keys exist in config)

### `crypto/oauth`
- [ ] **H** Refresh token flow
- [ ] **H** Authorization code flow (redirect, code exchange, state/PKCE)
- [ ] **H** Token endpoint handler
- [ ] **M** Redirect URI validation
- [ ] **L** Dynamic scope loading (currently hardcoded)

### `events`
- [ ] **H** Implement `WithCorrelationFromContext()` (TODO at line 110)
- [ ] **H** SNS publish helper
- [ ] **H** SQS consume / subscription helper
- [ ] **M** DLQ handling and retry policies
- [ ] **M** Event schema validation
- [ ] **L** Event versioning beyond hardcoded `"1"`

### `http/cors/middleware`
- [ ] **H** Full CORS implementation (currently a pass-through stub with a TODO comment)
- [ ] **H** Preflight (`OPTIONS`) handling
- [ ] **M** Configurable allowed origins, methods, headers

### `http/csrf/middleware`
- [ ] **H** `ValidateHeaderValue` currently returns `true` unconditionally — wire up real check
- [ ] **H** CSRF token generation
- [ ] **M** Token storage and retrieval (cookie + header double-submit)

### `http/headers/eval`
- [ ] **H** CSRF token header validation (stub / TODO)
- [ ] **M** Cookie validation (stub / TODO)
- [ ] **M** Tighten `Content-Length` default (currently 4096 — too small for most APIs)

### `http/idempotency/middleware`
- [ ] **H** Wire up ElastiCache storage (code is commented out)
- [ ] **H** Fix TTL parsing (`"s"` suffix parsing on line 82)
- [ ] **M** Thread-safe in-memory store for single-instance fallback

### `http/request/helpers`
- [ ] **H** Implement `GetPathParams` (currently returns empty map — placeholder)
- [ ] **H** Implement `IsValidAPIKey` (TODO comment on lines 166–175)
- [ ] **L** Multipart / form-data request building

### `http/sanitization/middleware`
- [ ] **H** Reduce false-positive rate on sanitization patterns
- [ ] **M** Preserve numeric precision when re-encoding JSON body
- [ ] **L** Confirm `req.SetPathValue` compatibility with minimum Go version target

### `http/context/middleware`
- [ ] **H** Client IP extraction (commented out on line 36)
- [ ] **M** Path params extraction (placeholder)

### `http/telemetry/middleware`
- [ ] **H** Re-raise panics after logging (currently swallows them)
- [ ] **M** Distributed trace propagation (trace ID in / out of headers)

### `logging/otel`
- [ ] **H** Implement `Init()` — currently an empty stub with a TODO comment
- [ ] **H** Wire up OpenTelemetry SDK (traces + metrics)
- [ ] **M** Connect telemetry middleware to otel spans

### `testing/chaos`
- [ ] **M** Implement fault injection (random errors, partial failures)
- [ ] **M** Latency injection (configurable delays per call)
- [ ] **L** Dependency blackout simulation

### `testing/performance`
- [ ] **M** Implement latency measurement utilities (`latency.go` is empty)
- [ ] **M** Percentile (p50/p95/p99) helpers
- [ ] **L** Throughput / RPS measurement

### `testing/moxtox`
- [ ] **H** Reset `req.Body` after reading in condition matching (currently consumed)
- [ ] **M** Support content types beyond JSON in `extractRequestConditions`
- [ ] **M** Hash collision handling for scenarios with identical conditions
- [ ] **L** Replace `fmt.Printf` debug output with the SDK logger

---

## General SDK Health

- [ ] **H** Idempotent request body reading across all middleware (body consumed once; subsequent reads fail)
- [ ] **H** Add `context.Context` timeouts / deadlines consistently across all AWS clients
- [ ] **H** Re-raise panics after logging in telemetry middleware (currently swallowed)
- [ ] **H** CI: coverage gate, lint (`golangci-lint`), race detector (`-race`) in test run
- [ ] **M** Fix `response.Bind()` — uses `json.Marshal` on `res.Body` instead of `io.ReadAll`
- [ ] **M** Normalize error return conventions (some return empty string on miss, others return error)
- [ ] **M** Typed config values (currently all strings)
- [ ] **M** Centralized SDK initialization with dependency order (each package currently initializes independently)
- [ ] **L** Config schema validation
- [ ] **L** Module versioning strategy (`v2` path) and CHANGELOG

---

## Planned: AWS Service Connectors

- [ ] **H** **SQS** — send/receive/delete messages, DLQ support
- [ ] **H** **SNS** — topic publish, subscription management
- [ ] **H** **Cognito** — user pool management, token validation, admin ops
- [ ] **H** **SES** — transactional email sending (templated + raw)
- [ ] **H** **CloudWatch** — metrics publishing, log group / stream management
- [ ] **M** **EventBridge** — rule-based event routing, put-events helper
- [ ] **M** **RDS (non-Aurora)** — PostgreSQL / MySQL client wrapper (connection pool, query helpers)
- [ ] **M** **Lambda** — invoke (sync + async), event source mapping
- [ ] **M** **Kinesis** — stream producer / consumer helpers
- [ ] **M** **ElasticSearch / OpenSearch** — index / search / bulk helpers
- [ ] **L** **CloudFront** — signed URL / signed cookie generation, cache invalidation
- [ ] **L** **IAM** — assume-role, policy evaluation helpers
- [ ] **L** **Pinpoint / SNS Mobile Push** — push notification helpers

---

## Planned: Payment Processor Connectors

- [ ] **H** **Stripe** — payment intents, subscriptions, refunds, webhooks
- [ ] **H** **PayPal** — orders, captures, refunds, webhooks
- [ ] **H** **Apple Pay** — session validation, payment token decryption
- [ ] **H** **Google Pay** — payment data decryption, tokenization
- [ ] **H** **Klarna** — session creation, order management, webhooks
- [ ] **M** **Afterpay / Clearpay** — checkout, order capture, refunds
- [ ] **L** **Square** — payments, orders, catalog, webhooks
- [ ] **L** **Braintree** — transactions, vault, webhooks

---

## Planned: Third-Party API Connectors

### Identity & Auth
- [ ] **H** **Auth0** — management API, token exchange, user ops
- [ ] **H** **Twilio Verify** — SMS / TOTP / email OTP

### Communication
- [ ] **H** **Twilio** — SMS, voice, messaging services
- [ ] **M** **Slack** — webhook posting, bot API
- [ ] **M** **PagerDuty** — incident creation, alert routing

### Observability
- [ ] **H** **Datadog** — metrics, logs, traces submission
- [ ] **L** **New Relic** — telemetry ingest

### Search & Data
- [ ] **H** **Persona** — identity verification (KYC)
- [ ] **L** **Google Maps / Places** — geocoding, address validation
