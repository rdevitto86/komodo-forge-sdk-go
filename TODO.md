# TODO

A running list of gaps, incomplete work, and planned additions. Each item is labeled **H** (high), **M** (medium), or **L** (low) priority and ordered within each section accordingly.

---

## `api/handlers/health`

> **Migration note (2026-06-07):** `komodo-auth-api` already wired `GET /health/ready` with a hand-rolled stopgap (`internal/api.HealthReadyHandler`, raw `func(context.Context) error` checks, first-failure-only `{"status","error"}` body, no result caching) ahead of this package shipping — see `apis/komodo-auth-api/internal/api/health.go`. It diverges from the spec below on checker shape (no `Name()`/`Checker` interface), failure reporting (single error vs. the full `{"failing": [...]}` list), and caching (none vs. TTL-cached `cachedResult` map). When this ships, plan a migration pass in auth-api: swap to `NewReadyHandler`/`Checker`/`CheckerFunc`, adopt the `{"failing": [...]}` body shape (downstream consumers parsing the current shape need updating too), and pick up TTL-cached checks. Also note auth-api currently only injects a Redis `Reachable` check — no JWKS checker — despite this TODO citing "JWKS endpoint reachability in auth-api" as the `CheckerFunc` worked example; confirm with auth-api whether that check should be added.

---

## `auth` package — consumer token verification

> Implements the local-verify side of the introspect-vs-denylist ADR. Consumers inject `auth.Verifier` rather than calling `POST /v1/oauth/introspect` directly. Phase 1 (JWKS-backed local verify) shipped in v0.14.2. Phase 2 adds a bloom-filter denylist when Redis is in the infra budget.

- [ ] **L** Phase 2: `auth/denylist.go` — `DenylistVerifier` wraps `JWKSVerifier`; after local verify, checks a bloom filter backed by Redis; background goroutine refreshes bloom from the revocation set on configurable cadence; `ErrTokenRevoked` sentinel; `DenylistConfig` adds `RedisClient db.API`, `RefreshInterval time.Duration`, `FalsePositiveRate float64`

---

## In-Progress / Stubs to Complete

### `db/sql` (was `aws/aurora`)
- [ ] **H** Implement wire-protocol SQL client (`client.go`, `errors.go` are still stubs — pgx + `database/sql`)
- [ ] **H** Connection pool configuration (max open/idle conns, lifetime)
- [ ] **H** Transaction support
- [ ] **M** Query builder helpers (select, insert, update, delete)

### `aws/dynamodb`
- [ ] **H** Retry logic for unprocessed items in batch write/delete
- [ ] **H** Transaction support (`TransactWriteItems`, `TransactGetItems`)
- [ ] **M** Export `ErrNotFound` sentinel — `getItem` currently wraps `fmt.Errorf("item not found")` with no exported type; callers (e.g. `komodo-auth-api/internal/clients.BannedCustomersClient`) resort to `strings.Contains` on the error message. Add `var ErrNotFound = errors.New("item not found")` and return it from `GetItem`/`GetItemAs` when the DynamoDB `ItemCollectionMetrics` response is empty. Update `BannedCustomersClient` string-match workaround once this ships.
- [ ] **M** Conditional expression helpers beyond raw strings
- [ ] **L** Consistent read flag on Scan
- [ ] **L** Projection expression support

### `db/redis` (was `aws/elasticache`)
- [x] **M** `Incr(ctx, key) (int64, error)` — atomic increment; added to `API` interface + `Client` in v0.14.1. Consumed by `komodo-auth-api` `IncrOTPAttempts`.
- [x] **M** `SetNX(ctx, key, value string, ttl int64) (bool, error)` — set-if-not-exists; added in v0.14.1. Returns true on write, false if key already existed. Unblocks atomic OTP cooldown, distributed locks, and the `api/idempotency` stub.
- [x] **M** `Exists(ctx, key string) (bool, error)` — key existence check without fetching the value; added in v0.14.1. Aligns `db/redis` with the `gcp/memorystore` API stub contract.
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
- [ ] **M** Add `Watch(ctx context.Context, interval time.Duration, keys []string, onChange func(map[string]string))` — background goroutine re-fetches the secret blob at `interval`, diffs against the previous snapshot, and calls `onChange` only when at least one value in `keys` has changed; supervised with `defer recover()` + restart loop (or `safego.GoEvery` once that ships); services currently call `sm.Close()` immediately after bootstrap which destroys the client — callers that want live rotation must keep the client alive and call `Watch` before the secrets are needed; auth-api needs this to pick up rotated JWT signing keys without an ECS restart
- [ ] **M** Pagination for batch secret retrieval
- [ ] **L** Support binary secrets

### AWS Service Stubs

All previously-empty AWS service packages were implemented in 0.14.0 — see `CHANGELOG.md`. Remaining follow-ups:

- [ ] **M** `aws/bedrock` — `ConverseStream` streaming via `bedrockruntime.ConverseStream` + AWS event-stream protocol. Add `ConverseStream(ctx, ConverseInput) (<-chan ConverseStreamEvent, <-chan error)` to `API`.
- [ ] **M** `aws/rds` — `time.Time` parameter helper (currently caller must format dates as strings). `TimeToField(t time.Time, hint types.TypeHint)` would reduce boilerplate.
- [ ] **M** `aws/rds` — `TypeHint` support on `ExecuteStatementInput.Parameters` (DATE, TIME, TIMESTAMP, UUID, JSON, DECIMAL) so Aurora can coerce string-encoded values correctly.
- [ ] **L** `aws/cloudwatch/metrics` — `GetMetricStatisticsPaged` for multi-day windows that exceed the 1440-datapoint single-call cap.
- [ ] **L** `aws/opensearch` — multi-AZ / VPC-mode endpoint mapping via `DomainStatus.Endpoints` (map). Currently only single-AZ `DomainStatus.Endpoint` is mapped.
- [ ] **L** `aws/ses` — templated email support (`SendTemplatedEmail`) — dropped from 0.14.0 to keep scope tight.
- [ ] **L** `aws/lambda` — streaming response invocations via `InvokeWithResponseStream`.
- [ ] **L** `aws/connect` — pagination control on `ListContactFlows` for callers with >1000 flows.

### GCP Service Stubs (Empty)

> Scaffolded 2026-05-18 to mirror AWS layout. Each package has a `Config`, `Client` struct, `API` interface, and stub methods that panic. `New()` returns `ErrNotImplemented` until the real client lands. Goal: callers swap providers by changing import path; method signatures stay identical to the AWS counterpart where the underlying semantics map cleanly. Divergences (Firestore vs DynamoDB key model, Pub/Sub vs SNS+SQS split, no SES equivalent) are called out per-package.

- [ ] **H** `gcp/gcs/` — Cloud Storage (parity with `aws/s3`): `GetObject`, `GetObjectAs`, `PutObject`, `DeleteObject`. Use `cloud.google.com/go/storage`. Tests via `fake-gcs-server` or storage emulator.
- [ ] **H** `gcp/firestore/` — Firestore (parity with `aws/dynamodb`): `GetItem`, `PutItem`, `UpdateItem`, `DeleteItem`, `Query`, `QueryAs`. Use `cloud.google.com/go/firestore`. **Divergence:** Firestore uses document IDs, not composite PK/SK — `BuildKey` is intentionally omitted. Decide whether composite-key callers map to a synthetic `id = pk + "#" + sk` or whether the SDK exposes a parallel `BuildPath` helper. Tests via Firestore emulator.
- [ ] **H** `gcp/pubsubpub/` — Pub/Sub publisher (parity with `aws/sns`): `Publish`. Use `cloud.google.com/go/pubsub`. Tests via Pub/Sub emulator.
- [ ] **H** `gcp/pubsubsub/` — Pub/Sub pull subscriber (parity with `aws/sqs`): `Receive`, `Ack`, `Nack`. Use `cloud.google.com/go/pubsub`. **Divergence:** Pub/Sub has no native FIFO `MessageGroupId` / `MessageDeduplicationId` — ordering keys are per-topic; document the gap. Tests via emulator.
- [ ] **H** `gcp/cloudfunctions/` — Cloud Functions / Cloud Run invoke (parity with `aws/lambda`): `Invoke`. Use `cloud.google.com/go/functions` or HTTP-trigger via authenticated `http/client`. Decide sync vs async semantics.
- [ ] **H** `gcp/secretmanager/` — Secret Manager (parity with `aws/secretsmanager`): `GetSecret`, `GetSecrets`. Use `cloud.google.com/go/secretmanager`. Distinguish "not found" via `ErrNotFound`. Add proper timeout `ctx` (don't replicate the `context.TODO()` bug from AWS).
- [ ] **H** `gcp/cloudlogging/` — Cloud Logging (parity with `aws/cloudwatch` logs side): `Write`, `WriteBatch`. Use `cloud.google.com/go/logging`.
- [ ] **H** `gcp/cloudmonitoring/` — Cloud Monitoring (parity with `aws/cloudwatch` metrics side): `PutMetric`, `PutMetrics`. Use `cloud.google.com/go/monitoring`. Decide custom-metric type prefix convention (`custom.googleapis.com/komodo/<name>`).
- [ ] **H** `gcp/vertexai/` — Vertex AI generative models (parity with `aws/bedrock`): `Invoke`, `InvokeStream`. Use `cloud.google.com/go/vertexai`. Map model IDs (`gemini-*`) to align with Bedrock model selection conventions in `komodo-support-api`.
- [ ] **H** `gcp/cloudsql/` — Cloud SQL (parity with `aws/aurora`): `DB()`, `Ping`, `Close`, connection pool config, transactions, query helpers. Use `cloud.google.com/go/cloudsqlconn` with `database/sql`. IAM auth flag.
- [ ] **H** `gcp/memorystore/` — Memorystore Redis (parity with `aws/elasticache`): `Get`, `Set`, `Delete`, `Exists`. Use `github.com/redis/go-redis/v9` with Memorystore connector. Configurable timeouts (don't replicate elasticache's 3s/2s hardcoded bug).
- [ ] **M** `gcp/vertexsearch/` — Vertex AI Search (parity with `aws/elasticsearch`): `Search`, `Index`, `Delete`. Use `cloud.google.com/go/discoveryengine`. **Note:** semantics differ significantly from Elasticsearch — managed retrieval vs raw inverted index; document tradeoffs and what features (custom mappings, aggregations) are not supported.
- [ ] **M** `gcp/dialogflow/` — Dialogflow CX / CCAI agents (parity with `aws/connect`): `DetectIntent`. Use `cloud.google.com/go/dialogflow`. Connect's flow-builder semantics don't map directly — document.
- [ ] **M** `gcp/ccaiinsights/` — Contact Center AI Insights (parity with `aws/contactlens`): `AnalyzeConversation`, `GetAnalysis`. Use `cloud.google.com/go/contactcenterinsights`.

### GCP — cross-cutting

- [ ] **H** Shared `gcp/internal/clientopts` (or top-level `gcp/clientopts`) for `option.ClientOption` assembly — every GCP client takes a credentials JSON path/blob, endpoint override (emulator), and project ID; centralize the option-builder to avoid 14× duplication of the same boilerplate, mirroring the AWS-side gap called out in the audit (`awsconfig.LoadDefaultConfig` called per client).
- [ ] **H** Provider-neutral interfaces — lift the `API` interfaces from `aws/<svc>` and `gcp/<svc>` into a shared package (e.g. `storage`, `queue`, `secrets`, `database`) so callers depend on the interface, not the concrete provider. Required to deliver on the "swap by import path" promise; otherwise consumers still hard-code `aws/s3.Client` or `gcp/gcs.Client`. Pair with the existing top-level **AWS client interfaces** TODO under General SDK Health.
- [ ] **M** `gcp/ses-equivalent` decision — there is no native GCP transactional email service. Either (a) document `connectors/sendgrid` as the GCP-region default, (b) wire `connectors/mailgun`, or (c) accept that email delivery is provider-agnostic (lives outside `gcp/`). Pick one before consumers start picking ad-hoc.
- [ ] **M** `gcp/dynamostreams` equivalent — Firestore change streams via `firestore.Listen` / Eventarc → Pub/Sub. Mirror the planned `aws/dynamostreams` package (see entry above) once that one's design lands.
- [ ] **L** README / docs — once stubs are filled in, add `gcp/README.md` with the AWS↔GCP mapping table and per-package divergences (currently embedded in each `client.go` doc comment).

### `config`
- [ ] **H** File-based config loading (YAML / JSON)
- [ ] **M** Multi-environment support (dev / staging / prod profiles)
- [ ] **M** Thread-safe `SetLevel` for log level changes
- [ ] **L** Change notification / listener hooks

### `security/bannedcustomers` (new)
- [ ] **M** Implement `security/bannedcustomers` package — `Client`, `Checker` interface (`IsBanned(ctx, email) (bool, error)`), `Config` (TableName, DynamoDB `dynamodb.API`), proactive `expires_at` TTL check, fail-open semantics documented. Used cross-service by auth-api, order-api, and payments-api. `komodo-auth-api/internal/clients.BannedCustomersClient` is a local copy that should be deleted and replaced with an import of this package once it ships.

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

### `codegen/templates`

Surfaced by the `komodo-auth-api` rollout (2026-05-17). The existing `client-with-responses.tmpl` only fires when consumers generate `client: true`; auth-api generates `models: true, client: false` and hand-writes both a thin `clients/{comms,user}.go` HTTP layer **and** a `paths.go` file of `Path<Operation>` consts. The header in those files literally says: *"When a custom oapi-codegen paths template is available, replace this file with generated output."* Standardize before more consumers copy the pattern.

- [ ] **M** **Path-constants template** (`paths.tmpl`) — emit a `paths.go` next to `types.gen.go` containing `const Path<OperationID> = "<path>"` for every operation in the spec. Templating-only: no runtime deps, no imports beyond the package declaration. Document the same "drop `user-templates` to deviate" pattern as the existing client template. This is the immediate ask — auth-api ships hand-written `internal/models/comms/paths.go` and `internal/models/user/paths.go` that should disappear on next regenerate.
  - Decide naming when `operationId` is absent (fall back to method + path-segments PascalCased).
  - Decide how to expose path params: bare template string (`"/v1/users/{id}"`) is simplest; a typed helper (`PathUser(id string) string`) is nicer but requires the consumer to import it. Lean simple — bare strings — and let consumers `fmt.Sprintf` or `strings.NewReplacer` if they want.
  - Add a `templates_test.go` case mirroring the existing one (parseable + Komodo additions divider present).
- [ ] **M** **Models-only preset config** — ship `codegen/oapi-codegen.models.yaml` as a documented base for consumers who want types + path constants but not a full generated client (auth-api's case: it talks to downstream services via a hand-written shared `HttpClient` wrapping `http/client`, not a per-service generated client). Today each consumer re-types the same `generate.models: true / client: false / skip-prune: true` block.
- [ ] **L** **README cross-link** — once `paths.tmpl` lands, update `codegen/README.md` to document both templates side-by-side and add a "which template do I want?" decision flow at the top (`generated client` vs `models + paths only`).
- [ ] **L** **Scaffolder command** — `scripts/add-client.sh <consumer-service> <provider-service>` that drops a starter `oapi-codegen.yaml` (correct relative path to the provider's `openapi.yaml`, both templates wired), runs `oapi-codegen`, and prints next-step guidance. Removes the copy-paste-an-existing-consumer pattern.

### `http/client` — service-to-service auth

Surfaced by `komodo-auth-api/internal/clients/user.go` and the inline `jwt.SignToken("komodo-auth-api", "komodo-auth-api", "komodo-apis:service", 30, nil)` call in `internal/api/otp.go`. Every consumer of a private endpoint will need to mint and attach a short-lived service JWT on each outbound call — currently hand-rolled per call site.

- [ ] **M** **`WithServiceAuth(serviceName, scope, ttl)` round-tripper option** — composes with the existing `http/client.Client`. Internally signs a service JWT via `crypto/jwt.SignToken`, caches it in-process keyed by `(name, scope)` with refresh ~10–20% before expiry, and sets `Authorization: Bearer <token>` on every outbound request. Auth-api's `clients/user.go` collapses from 35 lines of manual `http.NewRequestWithContext` + header-set + status-check + unmarshal into a 1-liner call against the generated client.
- [ ] **L** **`ServiceClient` helper** — thin wrapper that pairs `client-with-responses.tmpl`'s generated `New(baseURL)` with `WithServiceAuth`, so a consumer wires a downstream service in one constructor instead of stitching the round-tripper, base URL, and generated client together by hand.

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

#### Architecture — Multi-Frontend Split

> Today moxtox is a single package with HTTP middleware as its only surface. The goal is to split it into a pure matching engine plus multiple thin frontends. The engine becomes a library; frontends are thin wrappers that expose it as middleware, an `http.Client` drop-in, a standalone sidecar binary, and eventually a `database/sql` driver. This unblocks in-process, out-of-process, and DB-level mocking from a single scenario format.

- [ ] **H** `moxtox/engine` — extract all matching, storage, and scenario resolution logic into a standalone sub-package with no HTTP dependency; expose `Engine` struct with `Match(req MatchInput) (Scenario, bool)`, `Load(scenarios []Scenario)`, and `Reset()`; `MatchInput` carries method, path, headers, query, body — no `*http.Request` coupling; this is the prerequisite for every other frontend
- [ ] **H** `moxtox/middleware` — refactor current package to import `engine` rather than embed matching logic; `New(env, opts...)` returns an `http.Handler` wrapper; existing behavioral contract preserved; this replaces the current `InitMoxtoxMiddleware` once the engine extraction is done
- [ ] **H** `moxtox/client` — drop-in `http.Client` replacement backed by the engine locally (no network round-trip); implement as a custom `http.RoundTripper` that calls `engine.Match` instead of dialing; add recorder mode (`WithRecorder(dest string)`) that intercepts real responses and serializes them to YAML scenario files; primary use case: unit-testing connector code (Stripe, PayPal, AWS SDK) without a running server
- [ ] **M** `moxtox/server` — standalone binary wrapping the engine with an HTTP listener; accepts a scenario YAML directory on startup, exposes the mock responses on a configurable port, and serves an admin API (`POST /admin/scenarios`, `DELETE /admin/scenarios/:id`, `GET /admin/hits`); ships as a container image (see Pipeline section below); this is the sidecar target
- [ ] **L** `moxtox/db` — `database/sql` driver that resolves queries against scenario-matched rows stored in SQLite; implement `sql.Driver` and `sql.Conn` interfaces; scenario YAML declares table name, column names, and row sets keyed by a SQL condition pattern; returns matched rows without a real DB or sqlmock-style verbose expectations; long-tail killer feature — design the interface now, implement after sidecar mode has users

#### `moxtox/server` — Pipeline Operational Requirements

> These items apply specifically to the standalone server binary. Sub-second startup and a small container image are the gate on CI adoption. Everything else is table stakes for teams running it as a k8s sidecar or docker-compose service.

- [ ] **H** Health and readiness endpoints — `GET /healthz` returns `200 OK` immediately after bind; `GET /readyz` returns `200 OK` only after at least one scenario file has been successfully loaded; both return plain `{"status":"ok"}` JSON; required by k8s liveness/readiness probes and docker-compose `healthcheck` directives to gate test execution
- [ ] **H** Structured JSON logs to stdout — every request log line must include `request_id`, `method`, `path`, `matched_scenario` (scenario name or `"unmatched"`), `latency_ms`, and `status`; use `log/slog` JSON handler; pipeline runners scrape stdout and these fields make CI failures debuggable without shell access to the container
- [ ] **H** Graceful shutdown with hit report — on `SIGTERM`, finish in-flight requests (5s drain), then write a summary to stdout: scenarios loaded, per-scenario hit counts, unmatched request count and paths, passthrough count; format as a JSON object on a single line so CI log scrapers can parse it; this summary is the primary debugging artifact when a test run ends unexpectedly
- [ ] **H** Config from env vars and mounted files — twelve-factor: `MOXTOX_SCENARIO_DIR` (default `./scenarios`), `MOXTOX_PORT` (default `8080`), `MOXTOX_ENV` (default `test`), `MOXTOX_LOG_LEVEL` (default `info`), `MOXTOX_DETERMINISTIC` (default `false`); YAML scenario files mounted at `MOXTOX_SCENARIO_DIR`; no config file required for the binary itself — env vars only
- [ ] **H** Container image — `Dockerfile` using distroless `gcr.io/distroless/static` base; multi-stage build (Go builder → distroless); multi-arch manifest (`linux/amd64`, `linux/arm64`); target image size ≤10MB; publish to `ghcr.io/moxtox/moxtox:latest` and version tags; include `HEALTHCHECK` directive pointing at `/healthz`; heavy images kill pipeline adoption
- [ ] **M** Prometheus `/metrics` endpoint — expose `moxtox_scenario_hits_total{scenario="..."}` counter, `moxtox_unmatched_requests_total` counter, `moxtox_passthrough_requests_total` counter, `moxtox_request_duration_seconds` histogram; lets teams alert on "tests started hitting the real backend unexpectedly" — that alert catches misconfigured scenario files before they silently produce wrong test results
- [ ] **M** Deterministic mode — when `MOXTOX_DETERMINISTIC=true`: disable any random jitter in delay simulation, fix `time.Now()` to a configurable epoch (default `2024-01-01T00:00:00Z`), evaluate scenarios in declaration order (not map iteration order); required to eliminate flaky CI caused by ordering non-determinism; expose as a functional option on `engine.Engine` so middleware and client frontends can also use it
- [ ] **M** Record-from-real mode — when started with `--record --upstream=https://staging.api.example.com`, proxy all requests to the upstream and serialize each response as a YAML scenario file in `MOXTOX_SCENARIO_DIR`; on subsequent runs without `--record`, replay recorded responses; this is the Hoverfly/VCR pattern — it is how teams bootstrap scenario files without writing them by hand; document the workflow: `moxtox --record` in staging → commit YAML → CI replays

#### Go-to-Market

- [ ] **H** Docker-compose example for a typical Go service stack — `docker-compose.yml` showing: the Go service under test, a moxtox sidecar mocking a downstream API, a Postgres container, a `healthcheck` on moxtox gating service startup, and a one-shot test runner container that exits with the test suite's exit code; this is the demo that sells the sidecar mode; publish it as `examples/docker-compose-go-service/` in the moxtox repo; a team should be able to clone and `docker compose up --exit-code-from tests` and see it work
- [ ] **M** README with competitive positioning — document explicitly: native Go (no JVM, sub-second startup, ≤10MB image), same engine in-process or out-of-process (WireMock cannot do in-process for non-JVM services), eventual DB mocking; include a side-by-side feature table against WireMock focused on the 20% that covers component/integration tests for Go API teams; do not market feature parity with WireMock — market simplicity for the Go-shop use case

---

## Testing Strategy — SDK-Wide

> Chosen stack: **unit** `testing + testify` · **component/mocking** `go.uber.org/mock` (mockgen) + moxtox RoundTripper · **integration** `testcontainers-go` (LocalStack, Redis, Postgres, OpenSearch, GCP emulators) · **performance** native `testing.B` for hot paths + `k6` for moxtox/server · **contract/E2E** `Pact-go` · **chaos/resilience** `Toxiproxy` via testcontainers-go.
>
> Test tiers form an ordered, cumulative ladder `unit < component < integration < e2e < chaos`, selected by the `TEST_TIER` env var and gated by the SDK `testing/testutil` skip-helpers (`Component`, `Integration`, `E2E`, `Chaos`). `-short` overrides `TEST_TIER` and forces unit-only; an unset/unrecognized `TEST_TIER` defaults to `unit`. CI runs `TEST_TIER=component` on every PR; `integration` on merge to main; `e2e` and `chaos` on release tags only.

---

### Infrastructure & Tooling

- [x] **H** Adopt tier convention — ~~build-tag~~ env-var ladder shipped in `testing/testutil/tiers.go` (`TEST_TIER` + `Component`/`Integration`/`E2E`/`Chaos` skip-helpers, `-short` → unit-only, default unit); supersedes the rejected `//go:build <tier>` approach
- [ ] **H** Document the tier convention — define the five tiers (`unit`, `component`, `integration`, `e2e`, `chaos`) in `CONTRIBUTING.md` with a clear decision rule for which tier each test belongs to and the `TEST_TIER`/`testutil` usage; update CI matrix to run the appropriate subset per trigger (`TEST_TIER=component` on PR, `integration` on merge, `e2e`/`chaos` on release tags)
- [ ] **H** Makefile test targets — add `make test-unit`, `make test-component`, `make test-integration`, `make test-e2e`, `make test-chaos`, and `make test-all`; each target sets the correct `TEST_TIER` env var (or `-short` for unit), `-race` detector, and `-count=1` (disable test result caching for integration+); PR CI calls `test-unit` and `test-component` only
- [ ] **H** Coverage gate in CI — run `TEST_TIER=component go test -coverprofile=coverage.out ./...` and fail the PR if any implemented package (not a stub returning `ErrNotImplemented`) falls below 80% statement coverage; use `go tool cover -func` to report per-package; add `coverage.out` to `.gitignore` (currently committed — see audit finding)
- [ ] **H** Pin and document test dependencies — add `go.uber.org/mock`, `github.com/stretchr/testify`, `github.com/testcontainers/testcontainers-go`, `github.com/pact-foundation/pact-go`, and `github.com/shopify/toxiproxy/v2` to `go.mod` under a `tools.go` with `//go:build tools` guard so they do not pollute the SDK's runtime dependency graph; `mockgen` binary pinned in the same file
- [ ] **M** Shared test helpers — extend the existing `testing/testutil` package (which already holds the tier skip-helpers) with reusable fixtures: `MustMarshalJSON(t, v)`, `MustReadFixture(t, path)`, `AssertErrorIs(t, err, target)`, and a `FakeResponseWriter` that captures status+body; today each `_test.go` file re-implements these inline; centralizing reduces noise and makes test intent clearer
- [ ] **M** `mockgen` generation script — add `scripts/generate-mocks.sh` that runs `mockgen` against every `API` interface in `aws/*/client.go`, `gcp/*/client.go`, `db/*/client.go`, and `http/client/client.go`; output to `<package>/mock/mock_<package>.go`; wire into `scripts/generate.sh` so `go generate ./...` keeps mocks in sync with interface changes

---

### Unit Tests

> Pure logic, no I/O, no goroutines. Run on every commit. Should complete in under 5 seconds for the whole SDK.

- [ ] **H** `crypto/jwt` — no tests exist; this package handles token signing and is security-critical; add table-driven tests for `SignToken` (valid, expired TTL, missing key), `ValidateToken` (valid, tampered signature, expired, wrong audience), and `ParseClaims` (valid claims, malformed token, wrong key); these are separate from `security/jwt` tests and must cover the `crypto/` path independently
- [ ] **H** `crypto/oauth` — no tests exist; add tests for all exported functions; cover grant type validation, scope parsing, and error cases
- [ ] **H** `connectors/stripe/*` — no tests in any of the six sub-packages (`checkout`, `customer`, `payapple`, `paygoogle`, `paymentintent`, `refund`); add unit tests for request construction, field mapping, and error parsing for each package; these packages are stubs today but tests should be written against the defined interfaces so they fail loudly when implementation is added
- [ ] **H** `connectors/paypal/*` — no tests in any of the three sub-packages (`orders`, `payment-sources`, `refund`); same approach as Stripe above
- [ ] **H** `connectors/apple/*` and `connectors/google/*` — no tests in any connector sub-package; add unit tests for `auth` (token validation logic, redirect construction) and `maps` (request encoding, response parsing) for both providers
- [ ] **M** `api/errors` — existing tests cover happy path; add table-driven cases for all `Range*` constants confirming error code uniqueness across the full range map; a duplicated range registration should fail at test time, not silently at runtime
- [ ] **M** `api/sanitization` — add unit tests for the `sanitizeBody` allocation path with a `ContentLength`-sized pre-allocation (verify the audit finding fix); add fuzz target (`FuzzSanitize`) for the body sanitizer to catch unexpected panics on malformed JSON
- [ ] **M** `api/redaction` — add unit tests for `containsSensitiveKey` specifically covering the `map[string]struct{}` fast path after the audit-finding fix; add a test that asserts the redaction regex does not corrupt base64 content, UUIDs, or long numeric strings (documents the regression risk)
- [ ] **M** `rules/eval` — add table-driven tests for all condition operators; cover the `Strict` mode boundary (allow-with-warning vs deny-on-missing); verify `ErrRuleNotFound` vs validation errors are distinct
- [ ] **M** `security/jwt` — existing tests present; add tests for the `sync.Once` key-loading fix (concurrent first-callers must not race); use `go test -race` explicitly in the test comment
- [ ] **L** `events/event` — add tests for all event type constructors and the `Version` field default; cover JSON round-trip fidelity (marshal → unmarshal → marshal produces identical bytes)
- [ ] **L** `logging/runtime` — add a test that asserts `RedactingLogger.Handle` calls `Enabled` before cloning the record (verifies the audit finding fix and prevents the optimization from being reverted)

---

### Component Tests

> Package under test with all external I/O replaced by injected fakes or gomock mocks. No Docker, no network. Gated with `testutil.Component(t)` (`TEST_TIER=component`); run in PR CI, not on a plain `go test ./...` (which is unit-only by default).

- [ ] **H** `aws/*` — generate gomock mocks for all `API` interfaces via `mockgen`; write component tests for every client method covering: success path, AWS SDK error wrapping (`ErrNotFound`, `ErrConflict`), context cancellation, and credential selection logic (static vs LocalStack vs default chain); target packages: `bedrock`, `cloudwatch/logs`, `cloudwatch/metrics`, `connect`, `connect/contactlens`, `dynamodb`, `elasticache`, `lambda`, `opensearch`, `rds`, `s3`, `secretsmanager`, `ses`, `sns`, `sqs`
- [ ] **H** `db/*` — same mock + component test pattern for `db/redis`, `db/opensearch`, `db/sql`; `db/redis` component tests must cover all `API` methods including the recently added `Incr`, `SetNX`, `Exists`; `db/sql` component tests must cover transaction begin/commit/rollback once the stub is implemented
- [ ] **H** `http/client` — component tests for circuit breaker state transitions (closed → open → half-open → closed) using a fake `http.RoundTripper` that returns configurable errors; verify that 4xx responses do not trip the breaker (audit finding fix); verify that `traceparent` header is propagated on outbound requests (audit finding fix)
- [ ] **H** `auth/middleware` — component tests using a fake `Verifier`; cover: valid token passes through, expired token returns 401, revoked token returns 401 (when `DenylistVerifier` is implemented), missing `Authorization` header returns 401, malformed `Bearer` prefix returns 401; verify that full error detail is NOT sent to the client (audit finding fix — only generic message)
- [ ] **H** `connectors/stripe/*` and `connectors/paypal/*` — once stubs are implemented, use moxtox `RoundTripper` frontend to mock Stripe and PayPal sandbox HTTP responses; component tests should cover the same cases as the unit tests above but through the real HTTP client path rather than request construction alone; this is the primary moxtox dogfood use case for this SDK
- [ ] **M** `gcp/*` — same mock + component test pattern once stubs are implemented; generate gomock mocks for GCP API interfaces; cover: success, `ErrNotFound`, `ErrNotImplemented` (current stub behavior), context cancellation
- [ ] **M** `api/idempotency` — component test covering the TOCTOU race condition fix; run two concurrent requests with identical idempotency keys against the in-memory store; verify only one succeeds and the second receives the cached response; use `-race` flag
- [ ] **M** `api/ratelimit` — component test for the bucket evictor goroutine shutdown; verify `Stop()` (once added) drains the goroutine within a timeout; run with `-race`; also test the consolidated env var name fix so `RATE_LIMIT_BUCKET_TTL_SEC` and `BUCKET_TTL_SECOND` do not diverge again
- [ ] **M** `events/client` — component tests using a fake SQS API; cover: `Subscribe` worker pool processes messages concurrently (not serially), handler error triggers visibility extension not silent drop, `MaxBatch > 10` construction returns an error, `ctx.Done()` stops the subscriber cleanly without goroutine leak; run with `-race`
- [ ] **M** `api/cors` and `api/csrf` — component tests for the middleware once real implementations land (both are currently stubs); CORS: verify preflight OPTIONS returns correct headers for allowed/disallowed origins; CSRF: verify `ValidateHeaderValue` rejects requests with missing or tampered tokens
- [ ] **L** `api/middleware/chain` — component test composing 5+ middleware in order; verify execution order, short-circuit on rejection, and that each middleware sees the correct request state (headers set by prior middleware are visible)

---

### Integration Tests

> Real infrastructure via `testcontainers-go`. Require Docker. Gated with `testutil.Integration(t)` (`TEST_TIER=integration`). Run on merge to main only.

- [ ] **H** `aws/dynamodb` integration — LocalStack container (`localstack/localstack`); test `GetItem`, `PutItem`, `UpdateItem`, `DeleteItem`, `Query`, `Scan`, `BatchWrite`, `BatchDelete` against a real DynamoDB-compatible endpoint; cover `ErrNotFound` sentinel once implemented; verify `runParallel` context cancellation propagation (audit finding)
- [ ] **H** `aws/s3` integration — LocalStack; test `GetObject`, `PutObject`, `DeleteObject`, `HeadObject`, `ListObjects`; verify streaming get does not load full object into memory (audit finding fix); test pre-signed URL generation
- [ ] **H** `aws/sqs` + `aws/sns` integration — LocalStack; test full publish → receive → ack round-trip via `events/client`; verify dead-letter queue routing on repeated handler failure; verify worker pool concurrency (not serial)
- [ ] **H** `aws/secretsmanager` integration — LocalStack; test `GetSecret`, `GetSecrets` with real secret values; verify "not found" error path; verify proper `ctx` timeout (not `context.TODO()`)
- [ ] **H** `db/redis` integration — `redis:7-alpine` container; test all `API` methods including `Incr`, `SetNX`, `Exists`; verify TTL expiry behavior (set a key with 1s TTL, sleep, assert `Get` returns not-found); verify `SetNX` atomicity (two concurrent callers, assert exactly one wins)
- [ ] **H** `db/sql` integration — `postgres:16-alpine` container; test connection pool, `Ping`, `Query`, `Exec`, `Transaction` (commit + rollback) once stub is implemented; verify `context.Context` timeout cancels the query mid-flight
- [ ] **H** `db/opensearch` integration — `opensearchproject/opensearch:2` container; test `Index`, `Search`, `Delete`, `BulkIndex`
- [ ] **M** `aws/rds` integration — LocalStack Aurora-compatible endpoint; test `ExecuteStatement`, `BatchExecuteStatement`, field type helpers; verify `TypeHint` coercion for DATE/UUID/JSON fields
- [ ] **M** `aws/ses` integration — LocalStack; test `SendEmail` and `SendBulkEmail`; verify recipient list limits and error wrapping
- [ ] **M** `aws/cloudwatch/logs` + `aws/cloudwatch/metrics` integration — LocalStack; test `PutLogEvents` batch fan-out (verify parallel dispatch audit fix) and `PutMetric` / `PutMetrics`
- [ ] **M** `gcp/*` integration — GCP emulators once stubs are implemented: Pub/Sub emulator (`gcr.io/google.com/cloudsdktool/cloud-sdk`) for `pubsubpub`/`pubsubsub`; Firestore emulator for `firestore`; Bigtable emulator for future packages; document emulator startup in `CONTRIBUTING.md`
- [ ] **L** `events/client` integration — full SQS round-trip via LocalStack; verify exponential backoff on `sqs.Receive` error; verify `ChangeMessageVisibility` extension is called on slow handler

---

### Performance Benchmarks

> Native `testing.B` for SDK hot paths (benchmarks run with `-bench=.` and are unaffected by `TEST_TIER`). `k6` scripts for moxtox/server HTTP throughput. No Docker required for benchmarks.

- [ ] **H** `api/sanitization` benchmark — `BenchmarkSanitizeBody` with 1KB, 10KB, and 100KB JSON payloads; establish baseline before and after the audit finding fix (pre-allocate from `ContentLength`); target: no more than one allocation per 1KB of body; add as a CI-tracked benchmark so regressions surface on PR
- [ ] **H** `api/redaction` benchmark — `BenchmarkContainsSensitiveKey` with 10, 50, and 200 keys; verify `map[string]struct{}` lookup is O(1) and outperforms the pre-fix `[]string` linear scan by at least 5× at 50+ keys
- [ ] **H** `security/jwt` benchmark — `BenchmarkValidateToken` for the on-request hot path; verify median latency stays under 500µs on a warm key cache; this is the primary `auth/middleware` cost
- [ ] **M** `api/ratelimit` benchmark — `BenchmarkBucketLookup` at 1K, 10K, and 100K unique client keys; verify per-request overhead stays under 5µs; catches bucket map lock contention regressions
- [ ] **M** `http/client` circuit breaker benchmark — `BenchmarkCircuitBreakerDo` in closed state (happy path overhead) and open state (fail-fast overhead); closed-state overhead should be under 2µs relative to a bare `http.RoundTripper`
- [ ] **M** `db/redis` benchmark — `BenchmarkGet` and `BenchmarkSet` against a local Redis container (integration tag); establish latency baseline for cache-hit and cache-miss paths used by `auth/denylist` (once implemented)
- [ ] **L** `moxtox/server` k6 load test — `testing/moxtox/k6/smoke.js` sending 100 RPS for 30s against a locally running `moxtox/server` container; assert p99 match latency < 5ms, zero unmatched requests, zero errors; run as part of moxtox release validation (not PR CI); `testing/moxtox/k6/soak.js` at 500 RPS for 5 minutes to catch goroutine leaks and memory growth

---

### Contract / E2E Tests (Pact-go)

> Consumer-driven contracts between this SDK's generated adapter clients and the Komodo service providers. Gated with `testutil.E2E(t)` (`TEST_TIER=e2e`). Run on release tags. Require the provider services to be running or have published pacts.

- [ ] **H** Pact setup — add `github.com/pact-foundation/pact-go` to `tools.go`; add `make test-e2e` target; document the Pact workflow (consumer generates pact file, provider verifies it) in `CONTRIBUTING.md`; decide on a Pact Broker vs file-based exchange strategy (file-based is simpler for a monorepo arrangement where all services are local)
- [ ] **H** `api/adapters/v1/auth` consumer pact — once the adapter is generated, write a Pact consumer test that exercises `POST /v1/oauth/token`, `POST /v1/oauth/introspect`, and `POST /v1/oauth/revoke`; the pact file is committed alongside the generated client; `komodo-auth-api` verifies it in its own CI
- [ ] **H** `api/adapters/v1/communications` consumer pact — `SendOTP` call (`POST /v1/send/email`); critical path for `komodo-auth-api` OTP delivery; pact ensures the endpoint shape does not change without a coordinated SDK update
- [ ] **M** `api/adapters/v1/user` consumer pact — covers `GET /v1/users/:id`, `POST /v1/users`, `PATCH /v1/users/:id`; used by auth-api and order-api
- [ ] **M** `api/adapters/v1/payments` consumer pact — covers `POST /v1/payment-intents`, `POST /v1/refunds`, webhook shape verification
- [ ] **L** Remaining adapter pacts (`cart`, `shop-items`, `order`, `search`, `support`, `reviews`) — lower priority since fewer cross-service callers; add as adapters are generated and consumers are identified

---

### Chaos & Resilience Tests (Toxiproxy)

> Network fault injection via Toxiproxy spun up by `testcontainers-go`. Gated with `testutil.Chaos(t)` (`TEST_TIER=chaos`). Run on release tags only. Require Docker. Implement the `testing/chaos` package as a thin wrapper around the Toxiproxy Go client so tests inject faults with a one-liner.

- [ ] **H** `testing/chaos` package implementation — implement the currently-empty stub; wrap `github.com/shopify/toxiproxy/v2/client` with helpers: `NewProxy(t, upstream) *Proxy`, `proxy.AddLatency(ms, jitter)`, `proxy.LimitBandwidth(bytesPerSec)`, `proxy.CutConnection()`, `proxy.Reset()`; spin up a Toxiproxy server container via `testcontainers-go` at test suite startup; `t.Cleanup` removes the proxy; this is the foundation for all chaos tests
- [ ] **H** `http/client` circuit breaker chaos — use `testing/chaos` to inject 100ms latency then cut the connection entirely; verify the breaker opens after the configured threshold, returns `ErrCircuitOpen` without dialing, and recovers (half-open → closed) after the reset timeout; verify 4xx responses do not trip the breaker (audit finding regression guard)
- [ ] **H** `db/redis` chaos — inject connection drops mid-request via Toxiproxy; verify `Get`/`Set` return a wrapped error (not panic); verify the connection pool re-establishes after the fault clears; verify `auth/denylist` (once implemented) fails open (allows the request) under Redis unavailability, consistent with documented fail-open semantics
- [ ] **H** `events/client` subscriber chaos — inject latency exceeding the SQS visibility timeout; verify `ChangeMessageVisibility` extension is called before timeout; inject `sqs.Receive` errors; verify exponential backoff (audit finding) and no tight error loop hammering the endpoint
- [ ] **M** `auth/middleware` chaos — make the JWKS endpoint unavailable via Toxiproxy; verify `JWKSVerifier` returns an error and the middleware rejects the request with 401 (not 500); verify that a cached JWKS key set is used for N seconds after the endpoint goes down (document the grace window in `JWKSVerifier.Config`)
- [ ] **M** `aws/dynamodb` chaos — inject latency and packet loss between the client and LocalStack; verify `runParallel` context cancellation stops in-flight goroutines (audit finding fix); verify retry logic does not loop indefinitely on persistent failure; verify `BatchWrite` unprocessed items are retried
- [ ] **M** `aws/s3` chaos — inject bandwidth throttle to simulate slow upload; verify streaming `GetObject` does not buffer the full body into memory under a slow connection; verify pre-signed URL generation is unaffected (no network call)
- [ ] **L** `db/sql` connection pool chaos — exhaust the connection pool by holding `MaxOpenConns` connections open via Toxiproxy pause; verify new requests receive a context deadline error, not a panic or hang; verify pool recovers when connections are released

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

- [ ] **M** **EventBridge** — rule-based event routing, put-events helper
- [ ] **M** **Kinesis** — stream producer / consumer helpers
- [ ] **L** **CloudFront** — signed URL / signed cookie generation, cache invalidation
- [ ] **L** **Pinpoint / SNS Mobile Push** — push notification helpers

---

## Planned: GCP Service Connectors

> Stubs for each are already scaffolded in `gcp/` (see "GCP Service Stubs (Empty)" above). This section tracks additional GCP services that don't yet have a stub package — analogous to the AWS planned-connectors list. Add a stub directory before implementation work begins so signatures can be reviewed independently.

- [ ] **M** **BigQuery** — query execution, dataset/table management, streaming inserts; common analytics destination, no AWS equivalent in this SDK (Redshift / Athena were never added)
- [ ] **M** **Eventarc** — event routing from GCP services into Pub/Sub / Cloud Run / Cloud Functions; parallel to AWS EventBridge entry below
- [ ] **M** **Cloud Tasks** — durable task queue with scheduled execution; no direct AWS equivalent (closest: SQS delay queues + Step Functions)
- [ ] **M** **Cloud Scheduler** — cron-as-a-service for HTTP / Pub/Sub / App Engine targets; parallel to AWS EventBridge Scheduler
- [ ] **L** **Cloud KMS** — envelope encryption, key rotation; pairs with the planned `security/encryption` package
- [ ] **L** **Cloud CDN / Cloud Armor** — signed URL generation, WAF rule management; parallel to the planned AWS CloudFront entry
- [ ] **L** **Cloud Translation / Speech-to-Text / Text-to-Speech** — only if downstream consumers (support-api, communications-api) need them; otherwise defer

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

## Audit Findings — 2026-05-25

> Net-new gaps found during full SDK re-audit. Items already tracked in the 2026-05-12 section or package-specific sections above are excluded.

---

### Security

- [ ] **H** `auth/middleware.go:26` — JWT validation `err.Error()` sent verbatim to client in `WithDetail`; log full error internally and return a generic `"token validation failed"` string to callers
- [ ] **H** `api/request/utils.go:178` — `GetClientKey` trusts the first `X-Forwarded-For` value unconditionally, allowing IP spoofing that bypasses rate limiting and IP access control; strip client-supplied hops and read only from a configured proxy trust depth
- [ ] **H** `api/redaction/middleware.go:67` — `longTokenRE` (20+ char alphanumeric run) matches base64 content, UUIDs, and long numeric strings, corrupting legitimate request body fields passed to downstream handlers; restrict the regex to the logging/audit copy path only

### Latency

- [ ] **H** `aws/dynamodb/operations.go:84` — `runParallel` does not propagate context cancellation; inflight goroutines continue running to completion after the parent context is cancelled; add `ctx.Done()` select inside each goroutine
- [ ] **M** `api/request/builder.go:27` — `json.Marshal` result converted to `string` then passed to `strings.NewReader`, making two copies of the body; use `bytes.NewReader(jsonBytes)` directly
- [ ] **M** `aws/cloudwatch/logs/client.go:105` — `PutLogEvents` sends multiple batches serially in a `for` loop; fan out independent batches in parallel with bounded goroutines (same pattern as `dynamodb.runParallel`)
- [ ] **M** `events/client.go:100` — `time.After` in the `Subscribe` backoff loop allocates a new `time.Timer` per iteration and leaks it if `ctx.Done()` fires first; replace with `time.NewTimer` + `timer.Stop()` + drain
- [ ] **L** `api/telemetry/middleware.go:56` — `finish_time` captured with a second `time.Now()` call, inconsistent with the `time.Since(start)` latency value; capture end time once and derive both fields from it

### Performance

- [ ] **H** `api/sanitization/middleware.go:78` — `sanitizeBody` allocates three times on every POST/PUT/PATCH body (`io.ReadAll` → `json.Unmarshal` → `json.Marshal`) without using `ContentLength` for buffer sizing; pre-allocate read buffer from `req.ContentLength` when positive
- [ ] **M** `api/redaction/middleware.go:112` — `containsSensitiveKey` performs a linear scan over a `[]string` slice on every request key; convert `sensitiveKeys` to `map[string]struct{}` and compile substring patterns into a single `*regexp.Regexp` at init time
- [ ] **M** `logging/runtime/redaction.go:11` — `RedactingLogger.Handle` clones every `slog.Record` and iterates all attributes unconditionally before checking whether the level is enabled; call `rl.Handler.Enabled(ctx, rec.Level)` first and skip work when filtered
- [ ] **M** `api/headers/eval.go:67` — `isValidContentLength` reads `MAX_CONTENT_LENGTH` from `os.Getenv` inside a closure on every request; cache the parsed value once at middleware construction time

### Optimization

- [ ] **M** `api/ratelimit/ratelimiter.go:255` — `startBucketEvictor` reads `RATE_LIMIT_BUCKET_TTL_SEC` while `loadEnv()` reads `BUCKET_TTL_SECOND`; the two env var names are different so neither is authoritative — consolidate bucket TTL into `envCfg` and use a single canonical name
- [ ] **M** `http/client/client.go:78` — circuit breaker counts 4xx responses as upstream failures, tripping the breaker on client errors (400 Bad Request, 404 Not Found) that reflect caller mistakes, not service health; only count 5xx responses as failures
- [ ] **L** `api/ratelimit/middleware.go:19` — log messages built with string concatenation (`"rate limiter failing open for client: " + key`), allocating on every call; pass client key as a structured attribute
- [ ] **L** `http/websocket/websocket.go:6` — uses stdlib `log.Printf` for all logging, bypassing `logging/runtime` structured logger; replace with `logger.Error`/`logger.Info` so messages appear in CloudWatch JSON format

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

- [ ] **L** `coverage.out` (62KB) is committed to version control — add to `.gitignore`.
- [ ] **L** `pre-commit` hook script lives in the repo root alongside source files — move to `scripts/` or `.githooks/`.
- [ ] **L** `go.mod` includes `aws-lambda-go` and `aws-lambda-go-api-proxy` as direct dependencies for all consumers including Fargate services. Move Lambda adapter support behind a `//go:build lambda` tag or a dedicated `server/lambda` sub-package to slim non-Lambda binaries.
- [ ] **L** `api/redaction/redaction.go:5` lone `// TODO - move common code over from middleware to here` comment — resolve or delete.
