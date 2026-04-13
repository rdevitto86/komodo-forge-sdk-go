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

### `aws/dynamo`
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
- [ ] **H** `aws/rds/client.go` — empty stub
- [ ] **H** `aws/ses/client.go` — empty stub
- [ ] **H** `aws/sns/client.go` — empty stub
- [ ] **H** `aws/sqs/client.go` — empty stub

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

### `api/circuitbreaker`
- [ ] **H** Implement circuit breaker (`breaker.go` is empty stub)

### `events`
- [ ] **H** Implement `Publish()` (`client.go` is a stub — returns nil)
- [ ] **H** Implement `WithCorrelationFromContext()` (TODO at line 110)
- [ ] **H** SNS publish helper
- [ ] **H** SQS consume / subscription helper
- [ ] **M** DLQ handling and retry policies
- [ ] **M** Event schema validation
- [ ] **L** Event versioning beyond hardcoded `"1"`

### `http/errors`
- [ ] **M** Register `RangePromotions = 62` in `ranges.go` — claimed by `komodo-shop-promotions-api`; services currently use a local constant with a TODO comment pending this registration
- [ ] **M** Register `RangeWishlist = 32` in `ranges.go` — claimed by `komodo-user-wishlist-api`; same pattern

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

### `api/idempotency`
- [ ] **H** Implement DistributedCache with Redis/ElastiCache integration (currently a stub with TODO comments)
- [ ] **H** Wire up ElastiCache storage (code is commented out in middleware)
- [ ] **M** Thread-safe in-memory store for single-instance fallback

### `http/request`
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

### `http/client`
- [ ] **H** Configurable timeouts (connection, request, total) - currently uses infinite defaults
- [ ] **H** Retry logic with exponential backoff for transient failures
- [ ] **H** Request/response middleware pipeline (logging, auth, tracing)
- [ ] **H** Observability hooks (metrics, structured logging, distributed tracing)
- [ ] **H** Circuit breaker integration to prevent cascading failures
- [ ] **M** Rate limiting support

### `http/response`
- [ ] **M** Fix `Bind()` — uses `json.Marshal` on `res.Body` instead of `io.ReadAll`

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

### `scripts`

> Goal: reusable shell scripts consumable by any Go app or API. Each script must be self-contained, exit non-zero on failure, and work in both local and CI environments. No SDK-specific logic.

- [x] **H** `build.sh` — compile binary via `go build`; inject version, commit SHA, and build timestamp at link time via `-ldflags`; accept target `GOOS`/`GOARCH` as env vars for cross-compilation
- [x] **H** `test.sh` — run `go test ./... -race`; support `TEST_FLAGS` env var for passthrough args (e.g. `-short`, `-run <pattern>`); print pass/fail summary and exit non-zero on any failure
- [x] **H** `lint.sh` — run `golangci-lint run`; install lint binary if not found on `PATH`; respect `.golangci.yml` config if present; exit non-zero on any finding
- [x] **H** `audit.sh` — run `go mod verify`, `go vet ./...`, and `govulncheck ./...`; install `govulncheck` if not found; print a clear summary section for each tool; exit non-zero if any check fails
- [x] **H** `coverage.sh` — run tests with `-coverprofile=coverage.out`; generate HTML report via `go tool cover -html`; enforce a minimum coverage threshold (configurable via `COVERAGE_THRESHOLD` env var, default 80); print per-package breakdown
- [x] **M** `format.sh` — run `gofmt -w ./...` and `goimports -w ./...`; install `goimports` if not found; in CI mode (`CI=true`) run in check-only mode and exit non-zero if any files would be changed rather than modifying them
- [x] **M** `generate.sh` — run `go generate ./...`; optionally install common codegen tools (`mockgen`, `stringer`) if declared in a `tools.go` file; print which files were modified

#### Shared Script Standards
- [x] **M** Add a common `_lib.sh` helper sourced by all scripts — provides `log_info`, `log_error`, `require_tool` (checks PATH + installs if missing), and `check_go_version` (enforces minimum Go version)
- [x] **L** Add `CI` env var awareness to all scripts — stricter output (no color codes), check-only mode where applicable (format, generate), non-interactive installs
- [x] **L** Add usage/help output (`--help` flag) to each script documenting supported env vars and exit codes

---

### `testing/moxtox`

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
- [ ] **H** Remove `http/errors` import — replace `httpErr.SendError` and `httpErr.SendCustomError` calls with stdlib `http.Error` as the default; expose an injectable `ErrorHandler func(w http.ResponseWriter, r *http.Request, status int, code, message string)` so consumers can plug in RFC 7807, JSON:API, or any other error format
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

> Goal: move all per-service adapters out of `komodo-ecom/apis/<service>/pkg/v1/` and into this SDK. OpenAPI specs are the source of truth — types and HTTP clients are generated, not hand-written.

### Codegen pipeline

- [ ] **H** Add `scripts/generate-adapters.sh` — iterate over each Komodo service's `docs/openapi.yaml`, run `oapi-codegen` to emit types + HTTP client into `api/adapters/v{N}/<service>/`; check in generated output; CI step diffs generated code against spec and fails on mismatch
- [ ] **H** Add `tools.go` declaring `oapi-codegen` as a tracked Go tool dependency (`go install` friendly, pinned version)
- [ ] **M** Wire `generate-adapters.sh` into existing `scripts/generate.sh` so a single `go generate ./...` regenerates everything

### Komodo service adapter targets (migrate from `pkg/v1/`)

- [ ] **H** `api/adapters/v1/auth/` — generated from `komodo-auth-api/docs/openapi.yaml`; replaces `komodo-auth-api/pkg/v1/`
- [ ] **H** `api/adapters/v1/user/` — generated from `komodo-user-api/docs/openapi.yaml`; replaces `komodo-user-api/pkg/v1/`
- [ ] **H** `api/adapters/v1/payments/` — generated from `komodo-payments-api-rust/docs/openapi.yaml` (Rust service — no existing Go pkg/v1 to migrate)
- [ ] **M** `api/adapters/v1/cart/` — generated from `komodo-cart-api/docs/openapi.yaml`; replaces `komodo-cart-api/pkg/v1/`
- [ ] **M** `api/adapters/v1/shop-items/` — generated from `komodo-shop-items-api/docs/openapi.yaml`; replaces `komodo-shop-items-api/pkg/v1/`
- [ ] **M** `api/adapters/v1/order/` — generated from `komodo-order-api/docs/openapi.yaml`; replaces `komodo-order-api/pkg/v1/`
- [ ] **M** `api/adapters/v1/order-reservations/` — generated from `komodo-order-reservations-api/docs/openapi.yaml`; replaces `komodo-order-reservations-api/pkg/v1/`
- [ ] **M** `api/adapters/v1/search/` — generated from `komodo-search-api/docs/openapi.yaml`; replaces `komodo-search-api/pkg/v1/`
- [ ] **M** `api/adapters/v1/support/` — generated from `komodo-support-api/docs/openapi.yaml`; replaces `komodo-support-api/pkg/v1/`
- [ ] **M** `api/adapters/v1/communications/` — generated from `komodo-communications-api/docs/openapi.yaml`; replaces `komodo-communications-api/pkg/v1/`
- [ ] **M** `api/adapters/v1/reviews/` — generated from `komodo-reviews-api/docs/openapi.yaml`; replaces `komodo-reviews-api/pkg/v1/`
- [ ] **L** Add unversioned re-export at `api/adapters/<service>/` pointing to current stable version — consumers can import the unversioned path and stay on latest without code changes

---

## General SDK Health

- [ ] **H** Idempotent request body reading across all middleware (body consumed once; subsequent reads fail)
- [ ] **H** Add `context.Context` timeouts / deadlines consistently across all AWS clients
- [ ] **H** CI: coverage gate, lint (`golangci-lint`), race detector (`-race`) in test run
- [ ] **M** Normalize error return conventions (some return empty string on miss, others return error)
- [ ] **M** Typed config values (currently all strings)
- [ ] **M** Centralized SDK initialization with dependency order (each package currently initializes independently)
- [ ] **L** Config schema validation
- [ ] **L** Module versioning strategy (`v2` path) and CHANGELOG — include default re-export pattern: each package root re-exports from its current stable versioned subpackage so consumers import a single canonical unversioned path; older/newer versions remain importable via their versioned subpath (e.g. `http/middleware` re-exports from `http/middleware/v1`)

---

## Planned: AWS Service Connectors

- [ ] **H** **SQS** — send/receive/delete messages, DLQ support
- [ ] **H** **SNS** — topic publish, subscription management
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
