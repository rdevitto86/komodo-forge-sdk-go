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

### `aws/secretsmanager`
- [ ] **H** Replace `context.TODO()` with proper timeout context in `GetSecret`/`GetSecrets`
- [ ] **H** Distinguish "not found" from other errors
- [ ] **M** Pagination for batch secret retrieval
- [ ] **L** Support binary secrets

### AWS Service Stubs (Empty)
- [ ] **H** `aws/cloudwatch/` — directory exists but is empty
- [ ] **H** `aws/connect/client.go` — empty stub
- [ ] **H** `aws/contactlens/client.go` — empty stub
- [ ] **H** `aws/elasticsearch/client.go` — empty stub
- [ ] **H** `aws/lambda/client.go` — empty stub
- [ ] **H** `aws/ses/client.go` — empty stub

### `config`
- [ ] **H** File-based config loading (YAML / JSON)
- [ ] **M** Multi-environment support (dev / staging / prod profiles)
- [ ] **M** Thread-safe `SetLevel` for log level changes
- [ ] **L** Change notification / listener hooks

### `security/jwt`
- [ ] **H** Token revocation / JTI blacklist (revocation check is currently commented out)
- [ ] **H** Token refresh / rotation mechanism
- [ ] **M** Support for multiple concurrent key versions (JWKS-style)
- [ ] **M** Token introspection
- [ ] **L** Key pair generation helper (currently assumes keys exist in config)

### `security/oauth`
- [ ] **H** Refresh token flow
- [ ] **H** Authorization code flow (redirect, code exchange, state/PKCE)
- [ ] **H** Token endpoint handler
- [ ] **M** Redirect URI validation
- [ ] **L** Dynamic scope loading (currently hardcoded)

### `security/encryption`
- [ ] **H** Implement encryption package (`encryption.go` is empty stub)

### `http/client` — circuit breaker wiring
- [ ] **M** Wire `WithCircuitBreaker` into the following Komodo call sites:
  - **komodo-auth-api**: ElastiCache token revocation checks (`oauth_token_handler.go:173`)
  - **komodo-cart-api**: `shop-items-api` (product snapshots), `shop-inventory-api` (stock holds)
  - **komodo-support-api**: Anthropic Haiku LLM calls (`anthropic.go:39`)
  - **komodo-address-api**: External address validation provider (SmartyStreets/Google) — currently stubs (`address.go:50,59,77`)
  - **komodo-search-api**: Typesense search queries
  - **komodo-communications-api**: SendGrid/SES email, Twilio/SNS SMS
  - **komodo-shipping-api**: Carrier aggregator API (EasyPost/ShipStation)
  - **komodo-payments-api**: Stripe API calls (`payment_intents`, `refunds`)
  - **komodo-event-bus-api**: SNS publish calls (CDC Lambda and relay publisher)
  - **Cross-service calls**: cart-api ↔ inventory-api, order-api ↔ payments-api, order-api ↔ shipping-api, returns-api ↔ payments-api/inventory-api

### `security/hashing` (new)
- [ ] **H** Shared password/token hashing utility — standardize on Argon2id (preferred) or bcrypt across all Komodo services; expose `Hash(plaintext) (string, error)` and `Verify(plaintext, hash) (bool, error)`; replace ad-hoc hashing currently done per-service in `komodo-auth-api` and future `komodo-user-api` password storage

### `aws/dynamostreams` (new)
- [ ] **M** Generic DynamoDB Streams consumer/subscriber — beyond the single CDC Lambda in `komodo-event-bus-api`, services like `statistics-api`, `insights-api`, and `search-api` need to consume stream events for real-time aggregation and index sync; provide shard management, checkpointing, retry, and a handler callback interface so any service can subscribe without reimplementing the plumbing

### `events`
- [ ] **M** DLQ handling and retry policies
- [ ] **M** Event schema validation
- [ ] **L** Event versioning beyond hardcoded `"1"`

### `api/errors`
- [ ] **M** Register `RangePromotions = 62` in `ranges.go` — claimed by `komodo-shop-promotions-api`; services currently use a local constant with a TODO comment pending this registration
- [ ] **M** Register `RangeWishlist = 32` in `ranges.go` — claimed by `komodo-user-wishlist-api`; same pattern

### `api/cors/middleware`
- [ ] **H** Full CORS implementation (currently a pass-through stub with a TODO comment)
- [ ] **H** Preflight (`OPTIONS`) handling
- [ ] **M** Configurable allowed origins, methods, headers

### `api/csrf/middleware`
- [ ] **H** `ValidateHeaderValue` currently returns `true` unconditionally — wire up real check
- [ ] **H** CSRF token generation
- [ ] **M** Token storage and retrieval (cookie + header double-submit)

### `api/headers/eval`
- [ ] **H** CSRF token header validation (stub / TODO)
- [ ] **M** Cookie validation (stub / TODO)
- [ ] **M** Tighten `Content-Length` default (currently 4096 — too small for most APIs)

### `api/idempotency`
- [ ] **H** Implement DistributedCache with Redis/ElastiCache integration (currently a stub with TODO comments)
- [ ] **H** Wire up ElastiCache storage (code is commented out in middleware)
- [ ] **M** Thread-safe in-memory store for single-instance fallback

### `api/request`
- [ ] **H** Implement `GetPathParams` (currently returns empty map — placeholder)
- [ ] **H** Implement `IsValidAPIKey` (TODO comment on lines 166–175)
- [ ] **L** Multipart / form-data request building

### `api/sanitization/middleware`
- [ ] **H** Reduce false-positive rate on sanitization patterns
- [ ] **M** Preserve numeric precision when re-encoding JSON body
- [ ] **L** Confirm `req.SetPathValue` compatibility with minimum Go version target

### `http/context/middleware`
- [ ] **H** Client IP extraction (commented out on line 36)
- [ ] **M** Path params extraction (placeholder)

### `api/telemetry/middleware`
- [ ] **H** Re-raise panics after logging (currently swallows them)
- [ ] **M** Distributed trace propagation (trace ID in / out of headers)

### `http/client`
- [ ] **H** Configurable timeouts (connection, request, total) - currently uses infinite defaults
- [ ] **H** Retry logic with exponential backoff for transient failures
- [ ] **H** Request/response middleware pipeline (logging, auth, tracing)
- [ ] **H** Observability hooks (metrics, structured logging, distributed tracing)
- [ ] **M** Rate limiting support

### `api/response`
- [ ] **M** Fix `Bind()` — uses `json.Marshal` on `res.Body` instead of `io.ReadAll`

### `api/server` — opinionated Server wrapper
- [ ] **M** Add `server.Server` struct that embeds `*http.Server` and carries shared dependencies (logger, metrics emitter, downstream SDK adapter clients, redis/db handles). Consumers register handlers as methods on `*Server` and access deps via `srv.Comms.SendOTP(...)` etc. — matches enterprise patterns (F100 Go shops, Java/.NET muscle memory) and reduces per-handler constructor boilerplate when many handlers share the same dep set. Keep current `Run(*http.Server, ...)` entrypoint working — the new type wraps stdlib `*http.Server`, doesn't replace it. Document the trade-off vs. per-handler constructor injection in the package doc.

### `concurrency/safego` (new)
- [ ] **M** Add `safego.Go(ctx, name string, fn func(context.Context))` — wraps a goroutine with `defer recover()` + structured log of the panic + restart policy hook (caller-supplied or default no-op). For request goroutines, the existing telemetry middleware recover is sufficient; this helper is specifically for long-lived background workers (cache refreshers, queue subscribers, scheduled jobs) where an unrecovered panic would silently kill the worker and the service would keep running half-broken. Init goroutines should NOT use this — startup failures should crash the process to trigger container restart.
- [ ] **L** Add `safego.GoEvery(ctx, name, interval, fn)` — convenience for periodic background work (cache TTL refresh, metric flush). Same recover semantics.

### `testing/performance`
- [ ] **H** Implement latency measurement (`latency.go` is empty stub)
- [ ] **M** Percentile (p50/p95/p99) helpers
- [ ] **L** Throughput / RPS measurement

### `testing/chaos`
- [ ] **H** Implement fault injection (`chaos.go` is empty stub)
- [ ] **M** Latency injection (configurable delays per call)
- [ ] **L** Dependency blackout simulation

### `logging/otel`
- [ ] **H** Implement `Init()` — currently an empty stub with a TODO comment
- [ ] **H** Wire up OpenTelemetry SDK (traces + metrics)
- [ ] **M** Connect telemetry middleware to otel spans

---

### `testing/moxtox`

#### Open Source Governance (pre-release blockers)
- [ ] **H** Add Apache 2.0 `LICENSE` file — required before any public release; include copyright header (`Copyright (c) 2024 Moxtox Contributors`) in each source file
- [ ] **H** Set up new Proton Mail address + Git org for moxtox — keep it organizationally separate from Komodo; org name, repo URL, and contact email must be decided before publishing
- [ ] **H** Add Developer Certificate of Origin (DCO) — add `DCO` file to repo root (standard Linux Foundation text); wire up `dco` check in CI (e.g. `github.com/dco-check/action`) so every PR commit has a valid `Signed-off-by` trailer

#### Bug Fixes
- [ ] **H** Reset `req.Body` after reading in condition matching — use `io.NopCloser(bytes.NewBuffer(...))` to restore body so subsequent reads (e.g. middleware chain) are not broken
- [ ] **H** Fix quick mode hash mismatch — `buildHashLookupMap` hashes only the keys defined in config conditions, but `extractRequestConditions` hashes all headers/query/body; request hash will never match scenario hash for any request with extra headers
- [ ] **H** Cache `countTotalScenarios()` result at init time — currently called on every request in dynamic mode, causing a full mapping scan per request
- [ ] **H** Replace package-level `sync.Once` + global `config`/`allowMocks` vars with instance-scoped struct — global state prevents multiple moxtox instances and breaks parallel test suites

#### Core Features (v0.1.0 required)
- [ ] **H** Path parameter matching — support named segments (e.g. `/users/:id`, `/orders/:orderId/items/:itemId`) as a condition type, populated from URL path at match time
- [ ] **H** Transport-level mode — wrap `http.Client` via a custom `http.RoundTripper` so outbound calls (e.g. to Stripe, PayPal) can be mocked without a running server; this is the primary use case for connector testing
- [ ] **H** Response sequencing — allow scenarios to define an ordered list of responses so successive calls return different results (e.g. call 1 → 200, call 2 → 429, call 3+ → 503); essential for retry and circuit-breaker tests
- [ ] **M** Regex condition matching — allow condition values to be regex patterns (e.g. `Authorization: Bearer .*`, path segment matches); required for realistic header and token matching
- [ ] **M** Wildcard `*` condition value — simple glob match as a lighter alternative to regex for common cases (e.g. match any value for a required key)
- [ ] **M** Support content types beyond JSON in body condition matching — form-encoded (`application/x-www-form-urlencoded`), multipart, plain text
- [ ] **M** Hash collision handling — when two scenarios produce the same condition hash in quick mode, fall back to slice-based linear scan rather than silently dropping one scenario
- [ ] **M** Scenario `not` conditions — allow negated matching (e.g. match requests where header `X-Feature` is absent or body field `status` is not `"active"`)

#### Open Source Decoupling
- [ ] **H** Extract into a standalone module with its own `go.mod` — no imports from `komodo-forge-sdk-go`; only external dependency should be `gopkg.in/yaml.v3`
- [ ] **H** Remove `logging/runtime` import — define a `Logger` interface (`Info(msg string)`, `Error(msg string, err error)`, `Debug(msg string)`); default implementation wraps stdlib `log/slog` (Go 1.21+); consumers inject their own via functional option
- [ ] **H** Remove `api/errors` import — replace `httpErr.SendError` and `httpErr.SendCustomError` calls with stdlib `http.Error` as the default; expose an injectable `ErrorHandler func(w http.ResponseWriter, r *http.Request, status int, code, message string)` so consumers can plug in RFC 7807, JSON:API, or any other error format
- [ ] **H** Adopt functional options pattern — replace `InitMoxtoxMiddleware(env string, configPath ...string)` signature with `New(env string, opts ...Option) *Moxtox`; options include `WithLogger`, `WithErrorHandler`, `WithConfigPath`, `WithNoMatchHandler`, `WithDefaultDelay`
- [ ] **M** Make no-match behavior injectable — currently hardcodes `418 Teapot` + `"MOXTOX_001"` error code (SDK-specific format); default to a plain JSON `404` with a generic message; allow override via `WithNoMatchHandler`
- [ ] **M** Make config format pluggable — define a `ConfigLoader interface { Load(path string) ([]byte, error) }` with a default YAML implementation; allows consumers to source config from embedded files, S3, environment variables, etc.
- [ ] **L** Remove hardcoded `loggerPrefix` constant — make the log prefix configurable via `WithLogPrefix(prefix string)` option so consumers can namespace log output to match their service name

#### Quality & Reliability
- [ ] **M** Validate YAML config on load — return descriptive errors for missing required fields, unknown `performanceMode` values, and malformed scenario structures rather than silently falling back
- [ ] **M** Switch YAML internal parsing from `map[interface{}]interface{}` to `map[string]interface{}` — removes unsafe type assertions throughout `parseMapping`/`parseScenario`
- [ ] **L** Per-scenario response header merging — allow a scenario to extend (not replace) global default headers defined at the config root
- [ ] **L** File path resolution — resolve `scenario.File` relative to the config directory, not the process working directory, so mock files are portable across environments

#### Test Coverage
- [ ] **H** Tests for quick mode — verify hash lookup returns the correct scenario, and that the fix for hash mismatch works end-to-end
- [ ] **H** Tests for body condition matching — including the body-restore fix (verify body is readable by the next handler after condition evaluation)
- [ ] **M** Tests for dynamic mode threshold switching — verify mode is selected correctly at 10-scenario boundary
- [ ] **M** Tests for each condition type in isolation — body, query, headers, path params
- [ ] **M** Tests for priority ordering — verify higher-priority scenario wins when multiple conditions match
- [ ] **M** Tests for response sequencing — verify correct response is returned on each successive call
- [ ] **M** Tests for ignored routes — verify passthrough behavior
- [ ] **M** Tests for environment gating — verify mocks are disabled when env is not in `allowedEnvironments`
- [ ] **L** Tests for delay — verify `time.Sleep` is applied within tolerance
- [ ] **L** Tests for dynamic template rendering — verify `{{.body.key}}` substitution

---

## API Adapters — Komodo Services

> Goal: generate per-service adapters in this SDK from each Komodo service's OpenAPI spec. OpenAPI specs are the source of truth — types and HTTP clients are generated, not hand-written.

### Codegen pipeline

- [ ] **H** Add `scripts/generate-adapters.sh` — iterate over each Komodo service's `docs/openapi.yaml`, run `oapi-codegen` to emit types + HTTP client into `api/adapters/v{N}/<service>/`; check in generated output; CI step diffs generated code against spec and fails on mismatch
- [ ] **H** Add `tools.go` declaring `oapi-codegen` as a tracked Go tool dependency (`go install` friendly, pinned version)
- [ ] **M** Wire `generate-adapters.sh` into existing `scripts/generate.sh` so a single `go generate ./...` regenerates everything

### Komodo service adapter targets

- [ ] **H** `api/adapters/v1/auth/` — generated from `komodo-auth-api/docs/openapi.yaml`
- [ ] **H** `api/adapters/v1/user/` — generated from `komodo-user-api/docs/openapi.yaml`
- [ ] **H** `api/adapters/v1/payments/` — generated from `komodo-payments-api/openapi.yaml` (Rust service, no existing Go pkg/v1 to migrate)
- [ ] **M** `api/adapters/v1/cart/` — generated from `komodo-cart-api/docs/openapi.yaml`
- [ ] **M** `api/adapters/v1/shop-items/` — generated from `komodo-shop-items-api/docs/openapi.yaml`
- [ ] **M** `api/adapters/v1/order/` — generated from `komodo-order-api/docs/openapi.yaml`
- [ ] **M** `api/adapters/v1/order-reservations/` — generated from `komodo-order-reservations-api/docs/openapi.yaml`
- [ ] **M** `api/adapters/v1/search/` — generated from `komodo-search-api/docs/openapi.yaml`
- [ ] **M** `api/adapters/v1/support/` — generated from `komodo-support-api/docs/openapi.yaml`
- [ ] **H** `api/adapters/v1/communications/` — generated from `komodo-communications-api/docs/openapi.yaml`. **Bumped from M to H**: komodo-auth-api OTP delivery is blocked on this — the `handlers.CommsClient` interface and nil-tolerant wiring is already in place, just needs a concrete client. Auth-api expects `SendOTP(ctx, email, code, ttlSeconds) error` semantics; align the generated client (or a thin SDK-side wrapper around it) so consumers don't have to compose `POST /v1/send/email` + template_id themselves.
- [ ] **M** `api/adapters/v1/reviews/` — generated from `komodo-reviews-api/docs/openapi.yaml`
- [ ] **L** Add unversioned re-export at `api/adapters/<service>/` pointing to current stable version — consumers can import the unversioned path and stay on latest without code changes

---

## General SDK Health

- [ ] **H** Idempotent request body reading across all middleware (body consumed once; subsequent reads fail)
- [ ] **H** Add `context.Context` timeouts / deadlines consistently across all AWS clients
- [ ] **H** CI: coverage gate, lint (`golangci-lint`), race detector (`-race`) in test run
- [ ] **H** Instance-based design — every package that currently uses a package-level `sync.Once` + global vars (`moxtox`, and any AWS client init) must be refactored to an explicit struct with a constructor; eliminates global mutable state, enables parallel tests, and allows multiple configurations in one process
- [ ] **H** AWS client interfaces — define a Go interface for every AWS client (Dynamo, S3, SQS, SNS, SecretsManager, ElastiCache, etc.) so callers can depend on the interface, not the concrete type; required for testability and for `moxtox` transport-level mocking of outbound AWS calls
- [ ] **H** Versioning and release strategy — establish git tag conventions (`v0.1.0`, semver), generate a `CHANGELOG.md` (keep-a-changelog format), document `v2` import path strategy; include default re-export pattern: each package root re-exports from its current stable versioned subpackage so consumers import a single canonical unversioned path
- [ ] **M** Normalize error return conventions (some return empty string on miss, others return error)
- [ ] **M** Adopt Moxtox error format consistently — `moxtox` currently uses a custom `code + message` error shape; decide whether this becomes the SDK-wide error envelope or stays moxtox-local; document the chosen pattern and update all packages to match
- [ ] **M** Typed config values (currently all strings)
- [ ] **M** Centralized SDK initialization with dependency order (each package currently initializes independently)
- [ ] **L** Config schema validation

---

## Planned: AWS Service Connectors

- [ ] **H** **SES** — transactional email sending (templated + raw)
- [ ] **H** **CloudWatch** — metrics publishing, log group / stream management
- [ ] **M** **EventBridge** — rule-based event routing, put-events helper
- [ ] **M** **RDS (non-Aurora)** — PostgreSQL / MySQL client wrapper (connection pool, query helpers)
- [ ] **M** **Lambda** — invoke (sync + async), event source mapping
- [ ] **M** **Kinesis** — stream producer / consumer helpers
- [ ] **M** **ElasticSearch / OpenSearch** — index / search / bulk helpers
- [ ] **L** **CloudFront** — signed URL / signed cookie generation, cache invalidation
- [ ] **L** **Pinpoint / SNS Mobile Push** — push notification helpers

---

## Planned: Payment Processor Connectors

- [ ] **H** **Stripe** — payment intents, subscriptions, refunds, webhooks
- [ ] **M** **Stripe — payment plans / installments** — installment schedule creation, per-installment charge execution, plan cancellation, and webhook events (`payment_plan.created`, `installment.paid`, `installment.failed`); complements subscription billing
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

### Shipping & Logistics
- [ ] **M** **EasyPost** — label generation (inbound + outbound), shipment creation, tracking events, carrier-agnostic API wrapper
- [ ] **L** **ShipStation** — order import, label generation, shipment status; alternative aggregator if EasyPost is not selected
- [ ] **L** **USPS / UPS / FedEx direct** — raw carrier APIs if aggregator is not used; each behind the same `ShippingProvider` interface so `komodo-shipping-api` can swap carriers without code changes

### Search & Data
- [ ] **H** **Persona** — identity verification (KYC)
- [ ] **L** **Google Maps / Places** — geocoding, address validation

---

## Audit Findings — 2026-05-12

> New gaps, correctness bugs, and performance issues found during full SDK audit. Moxtox and test coverage items are excluded (tracked separately).

---

### Critical correctness / security bugs

- [ ] **H** `api/request/parser.go:124-160` `GetClientType` decodes JWT payload **without verifying signature** and uses the result to exempt requests from CSRF/idempotency enforcement — any browser can forge `client_type:"api"` and bypass CSRF. Derive client type solely from context populated by `AuthMiddleware` after `ValidateAndParseClaims`.
- [ ] **H** `http/websocket/websocket.go:13` `CheckOrigin` returns `true` unconditionally — cross-site WebSocket hijacking. Require a configurable origin allowlist; no default-allow.
- [ ] **H** `api/headers/eval.go:20` `case "authorization": return jwt.ValidateToken(val)` passes the literal `"Bearer …"` string to a validator that expects the bare token — authorization header validation fails every request. Strip `"Bearer "` prefix before calling `ValidateToken`.
- [ ] **H** `security/jwt/jwt.go:34-63` `keysInitialized` is read outside the lock before write-lock is taken — data race on concurrent first callers. Replace with `sync.Once` for key loading.
- [ ] **H** `api/idempotency/idempotency.go` `LocalCache.Store` ignores TTL — entries never expire; map grows unbounded indefinitely. `Check`+`Set` is non-atomic (TOCTOU): two concurrent identical requests can both pass. Use SETNX semantics with a per-entry expiry sweep.
- [ ] **H** `api/csrf/middleware.go:31` delegates to `headers.ValidateHeaderValue("X-Csrf-Token", …)` which returns `true` unconditionally — CSRF protection is effectively off. Wire real token validation.
- [ ] **H** `api/sanitization/middleware.go:127-138` runs `html.EscapeString` on **all** header, query, and body strings — destructively mutates legitimate user input (`&`, `<`, `>` in URLs, JSON values, tokens). Scope to display-only fields or remove from middleware; do not silently rewrite payloads.
- [ ] **H** `api/telemetry/middleware.go:22-37` `recover()` swallows panics; also attempts `SendError` to the outer writer after a partial body write may have already occurred, and only sends if `status == 0`. Log the panic with stack trace via `debug.Stack()` and re-panic so the runtime handler can respond cleanly.

### Latency / performance

- [ ] **H** `aws/dynamodb/query.go:100-117, 181-198` `QueryAll`/`ScanAll` accumulate all items into a single in-memory slice with no limit. Add a streaming iterator (`Next() bool` / `Item()`) and a `MaxItems` guard to prevent OOM on large tables.
- [ ] **H** AWS clients (`dynamo`, `s3`, `sns`, `sqs`, `secretsmanager`, `elasticache`) each call `awsconfig.LoadDefaultConfig` independently — a service wiring 5+ clients pays 5+ IMDS/STS resolutions on cold start. Provide a shared `aws.Config` factory or accept an `aws.Config` in each `New()`.
- [ ] **M** `api/redaction/redaction.go:9-14` PII regex with multiple alternation groups runs on every log key/value pair. Benchmark; consider splitting into anchored sub-expressions or a string-scan fast path.

### Design / API issues

- [ ] **H** `aws/aurora`, `aws/lambda`, `aws/ses`, `aws/cloudwatch`, `aws/contactlens`, `aws/connect`, `aws/elasticsearch`, `aws/bedrock` — `New()` returns a non-nil `*Client{}` whose methods immediately `panic`. A caller has no way to know the client is unusable. Return `(nil, ErrNotImplemented)` from `New()` until real implementations land.
- [ ] **H** `api/request/builder.go:51` `FromRequest` passes the original `req.Body` reader to a new request — the body is consumed on first read; a second request reads nothing. Buffer the body and set `GetBody` on both.
- [ ] **H** `events/client.go:84-117` `Subscribe` processes messages serially — one slow handler stalls the entire queue. Add a bounded worker pool (`Concurrency int` on `SubscriberConfig`). On `sqs.Receive` error, no backoff — tight error loop hammers SQS on transient failure; add exponential backoff.
- [ ] **H** `events/client.go` on handler error, relies entirely on the queue's static visibility timeout with no `ChangeMessageVisibility` for explicit backoff or retry extension. Add configurable visibility extension during processing and on failure.
- [ ] **H** `rules/middleware.go:27-32` returns `400 BadRequest` when **no rule is defined** for a path — every unconfigured endpoint returns 400. Default should be allow-with-warning; add a `Strict bool` option to opt into deny-on-missing.
- [ ] **M** `api/normalization/normalization.go:52-55` writes to `req.RequestURI` for inbound server requests — this field is supplied by the runtime and rewriting it can confuse adapters (e.g. `aws-lambda-go-api-proxy`). Remove.
- [ ] **M** `api/normalization/normalization.go:68-78` switch cases for `"sort"`, `"asc"`, `"desc"` normalize a query **value** called literally "sort" — these look like key names, not values. Clarify intent or remove.
- [ ] **M** `api/errors/responses.go:39-47` calls `wtr.WriteHeader` before encoding the body; encode errors go undetected leaving a truncated response. Marshal to a buffer first, set `Content-Length`, then `Write` once.
- [ ] **M** `api/errors/responses.go:45` reads `X-Request-ID` from the request header — absent if `RequestIDMiddleware` hasn't run yet (e.g. early panic). Fall back to `req.Context().Value(REQUEST_ID_KEY)`.
- [ ] **M** AWS `New()` constructors all call `awsconfig.LoadDefaultConfig(context.Background(), …)` — an IMDS or STS hang stalls process startup indefinitely. Accept a `ctx` parameter or apply a bounded default deadline (e.g. 5s).
- [ ] **M** `api/middleware/exports.go:30` `ClientTypeMiddleware` is an alias for `ClientSourceMiddleware` — appears to be an unfinished rename. Remove the alias or complete the rename across all call sites.
- [ ] **M** `crypto/jwt` / `security/jwt` and `crypto/oauth` / `security/oauth` are duplicated import paths to the same implementations. Pick one canonical path per type, hard-deprecate the aliases, and delete once all services migrate. (`server` / `api/server` resolved: consolidated to `api/server`.)
- [ ] **L** `connectors/{apple,google,paypal,stripe}/*` are all single-line empty package declarations — indistinguishable from implemented packages. Remove or mark explicitly as stubs in this file.

### State / lifecycle

- [ ] **H** `security/jwt` (cached key globals), `idempotency` (`defaultStore`), `rules` (`ruleMap`, `patternRoutes`, `loadOnce`), `api/ratelimit` (`buckets`, `rps`, `burst`, `ecClient`), and `logging/runtime` (`slogger`, `initOnce`) all use package-level globals — makes multi-config usage and parallel tests impossible. Provide constructor-based instances for all of these (consistent with the existing global-state TODO that names only moxtox).
- [ ] **M** `api/ratelimit/ratelimiter.go:198-219` background bucket-eviction goroutine is started via `sync.Once` with no shutdown channel — goroutine leaks on test teardown and Lambda warm-cycle cleanup. Add a `Stop()` method or accept a `ctx` for cancellation.
- [ ] **M** `events/client.go:69` `NewSubscriber` silently clamps `MaxBatch > 10` to 10. Return a validation error or at minimum log a warning during construction.

### Observability

- [ ] **H** `http/client.Client.Do` does not propagate `traceparent`/`tracestate` headers or instrument with `httptrace.ClientTrace` — outbound requests are invisible to distributed tracing. Add trace propagation alongside the existing `logging/otel` work.
- [ ] **M** `api/telemetry/middleware.go` logs a flat `map[string]any` per request via `fmt.Errorf` wrapping — wastes structured logging. Use `slog` attributes directly on the logger call.
- [ ] **M** No metrics emission anywhere (request counts, circuit-breaker state changes, rate-limit denials, DynamoDB retry counts). Define a minimal `Metrics` interface (`Counter`, `Histogram`) with a no-op default; wire into client, middleware, and breaker.

### Cleanup / housekeeping

- [ ] **L** `aws/aurora/errors.go` is 0 bytes (empty file) — remove or populate.
- [ ] **L** `coverage.out` (62KB) is committed to version control — add to `.gitignore`.
- [ ] **L** `pre-commit` hook script lives in the repo root alongside source files — move to `scripts/` or `.githooks/`.
- [ ] **L** `go.mod` includes `aws-lambda-go` and `aws-lambda-go-api-proxy` as direct dependencies for all consumers including Fargate services. Move Lambda adapter support behind a `//go:build lambda` tag or a dedicated `server/lambda` sub-package to slim non-Lambda binaries.
- [ ] **L** `api/redaction/redaction.go:5` lone `// TODO - move common code over from middleware to here` comment — resolve or delete.
