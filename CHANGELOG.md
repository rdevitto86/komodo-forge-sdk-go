# Changelog

All notable changes to the Komodo Forge SDK are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.11.0]

### Added

- **`api/adapters/` — SDK adapters for Komodo internal services.** New top-level package housing outbound HTTP clients for sibling services. Cross-cutting conventions documented in `api/adapters/README.md`:
  - **Per-client API version.** `NewClient(baseURL string, ver int) (*Client, error)` builds URLs as `baseURL + "/v" + ver + path`. Constructor rejects empty `baseURL` and any `ver` outside the package-level `supportedVersions` set. One process can hold multiple `*Client` values targeting different versions of the same service for rolling migration.
  - **Per-client base URL.** Fixed at construction; per-call override is intentionally not supported. Canary / blue-green routing belongs at the LB / service-mesh layer.
  - **Two-layer surface.** Hand-curated typed methods (e.g. `comms.SendOTP`) for high-level operations + `Raw() *httpc.Client` escape hatch for routes not yet covered. Adapters stay thin — retry / timeout / circuit-breaker remain in `http/client`.
  - **Hand-curated typed-method registry.** Typed methods are added per consuming-service demand, not generated. Codegen (when it lands) emits low-level types + raw HTTP calls; the typed layer on top is the deliberate consumer surface.
- **`api/adapters/v1/communications/` — reference implementation.** Full typed surface with `SendEmail` and `SendOTP` (encapsulates the `"otp-request"` template ID). Unblocks `komodo-auth-api` OTP delivery — `handlers.CommsClient` satisfied by `*comms.Client`.
- **10 service adapter scaffolds** following the reference template: `api/adapters/v1/{auth,user,payments,cart,shop-items,order,order-reservations,search,support,reviews}/`. Each provides constructor + `Raw()` + `--- Typed surface ---` marker; typed methods are added one at a time as consuming services need them.

### Changed

- **Repository layout — split `/http` into transport vs. API layer.** `/http` now contains only protocol primitives (`client`, `context`, `websocket`); everything server-side moved to `/api`:
  - `http/server/` → `api/server/`
  - `http/handlers/` → `api/handlers/`
  - `http/middleware/` → `api/middleware/`
  - `http/cors/` → `api/cors/`
  - `http/csrf/` → `api/csrf/`
  - `http/headers/` → `api/headers/`
  - `http/sanitization/` → `api/sanitization/`
  - `http/normalization/` → `api/normalization/`
  - `http/ratelimit/` → `api/ratelimit/`
  - `http/redaction/` → `api/redaction/`
  - `http/telemetry/` → `api/telemetry/`
  - `http/ipaccess/` → `api/ipaccess/`
  - `http/errors/` → `api/errors/`
  - `http/request/` → `api/request/`
  - `http/response/` → `api/response/`
  - `idempotency/` → `api/idempotency/`
  - Mental model: `/http` = "how bytes move"; `/api` = "how Komodo services expose and consume APIs." All `git mv` to preserve history; all import paths updated repo-wide.
- **`server` / `api/server` deduplication.** Root `/server/` (real implementation: `Run`, `InitAndServe` with AWS Lambda detection) consolidated into `api/server/`, replacing the re-export shim previously at `http/server/`. The duplicate import-path pair flagged in the 2026-05-12 audit is resolved.
- **`TODO.md` / `README.md`** — package paths updated to the new `api/...` locations; cross-cutting adapter convention items checked off.

---

## [0.10.4]

### Changed

- **`aws/dynamodb` — package renamed** (`aws/dynamo/` → `aws/dynamodb/`) — directory and package declaration renamed from `dynamo` to `dynamodb` to match the official AWS service name. Import path is now `github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamodb`.
- **`README.md`** — corrected `http/client` option list: `WithTimeout` (non-existent) replaced with `WithCircuitBreaker`; updated `aws/dynamodb` import path.
- **`TODO.md`** — removed completed `api/circuitbreaker` stub items (circuit breaker is fully implemented in `http/client`); retained the Komodo service wiring item under `http/client — circuit breaker wiring`.

---

## [0.10.3]

### Changed

- **Codebase-wide error message cleanup** — removed service-name prefixes (`"sqs: ..."`, `"events: ..."`, etc.) from all `fmt.Errorf` calls and `logger` calls across `aws/sqs`, `aws/sns`, `aws/dynamodb`, `aws/elasticache`, `aws/secretsmanager`, `events`, `http/ratelimit`, and `rules`. Messages now read as actions (`"failed to receive message: ..."`) rather than repeating the package name.
- **`Makefile` — `release` target** — `make release` now reads `VERSION`, creates the git tag, and pushes it to origin in one step.
- **`README.md`** — corrected package paths, API signatures, and package inventory to match current SDK state.
- **`CHANGELOG.md`** — corrected version entries to accurately reflect what shipped in each tag.

---

## [0.10.1]

### Performance

- **`rules` — compile regex patterns at config load** (`rules/models.go`, `rules/parser.go`, `rules/eval.go`)
  - Replaced anonymous struct types in `Headers`, `PathParams`, `QueryParams`, and `Body` with a named `FieldSpec` struct carrying an unexported `compiled *regexp.Regexp` field.
  - Added `compileRulePatterns()` called once after YAML parse — all non-empty `Pattern` fields are compiled at startup. Invalid patterns now fail fast at load time rather than silently per-request.
  - `areValidHeaders`, `areValidPathParams`, and `areValidQueryParams` use `spec.compiled` directly instead of calling `regexp.Compile` on every inbound request (the single largest middleware latency reduction).
  - Replaced O(n²) bubble sort for route pattern specificity with `sort.SliceStable`.

- **`http/client` — tuned transport and per-host circuit breaker** (`http/client/client.go`, `http/client/circuitbreaker.go`)
  - Exported `DefaultTransport` (`*http.Transport`): `MaxIdleConnsPerHost: 20`, `MaxIdleConns: 100`, `IdleConnTimeout: 90s`, `TLSHandshakeTimeout: 10s`, `ResponseHeaderTimeout: 10s`, dial `Timeout: 5s`, `KeepAlive: 30s`. Previous default was Go's built-in transport with `MaxIdleConnsPerHost: 2`.
  - Added `WithTransport(http.RoundTripper) Option` so callers can supply a custom transport (TLS overrides, test transports).
  - Circuit breaker replaced single global `sync.Mutex` + `map[string]*breakerState` with `sync.Map` keyed by host, each entry with its own `sync.Mutex`. Parallel calls to different hosts no longer contend.
  - Added `MaxHosts int` to `Config`: when the cap is exceeded, new hosts bypass the breaker (fail-open guard against unbounded map growth).
  - Added `Prune()` method to remove closed, zero-failure entries on caller demand.

- **`aws/elasticache` — forward caller context** (`aws/elasticache/client.go`)
  - `Get`, `Set`, and `Delete` now accept `ctx context.Context` as first parameter (breaking change).
  - `API` interface updated to match.
  - A 2-second fallback deadline is applied only when the caller's context carries no deadline, preserving cancellation and distributed trace propagation.

- **`http/ratelimit` — eliminate per-request env lookups and fix data race** (`http/ratelimit/ratelimiter.go`)
  - `ENV` and `BUCKET_TTL_SECOND` environment variables are now parsed once at startup via `sync.Once` and stored in an `atomic.Pointer[envCfg]`. Previously read and parsed on every `Allow` call.
  - `rps` and `burst` package vars replaced with `atomic.Pointer[rlCfg]`. `LoadConfig` atomically stores a new config; `loadCfg` atomically loads — eliminates the data race between concurrent `LoadConfig` writes and `allow`/`retryAfter` reads.
  - Added `ResetForTesting()` to reset all atomic and `sync.Once` state between tests; updated test suite to use it.

- **`aws/dynamodb` — parallel batch operations** (`aws/dynamo/client.go`, `aws/dynamo/operations.go`)
  - Added `MaxConcurrentBatches int` to `Config` (default 5 when 0; set to 1 to restore serial behaviour).
  - `batchGetItems`, `batchWriteItems`, and `batchDeleteItems` dispatch 25-item chunks in parallel using goroutines bounded by a semaphore channel. Results for `batchGetItems` are pre-allocated by chunk index and merged in order.
  - Single-chunk inputs skip the goroutine overhead entirely.

- **`aws/secretsmanager` — in-process TTL cache** (`aws/secretsmanager/client.go`)
  - Added `CacheTTL time.Duration` to `Config` (default 5 minutes; negative value disables caching).
  - `GetSecret` checks the cache under `RLock` before issuing an API call; stores the result on miss.
  - `GetSecrets` caches the raw JSON blob at the batch path key so the AWS API is called at most once per TTL window regardless of how many individual key lookups occur.
  - Cache entries are checked for expiry on read; no background goroutine required.

### Changed

- **`aws/elasticache` API** — `Get(key string)`, `Set(key, value string, ttl int64)`, and `Delete(key string)` signatures changed to `Get(ctx, key)`, `Set(ctx, key, value, ttl)`, `Delete(ctx, key)`. Callers must pass a context.
- **Codebase-wide style pass** — removed redundant doc-comment prefixes from exported types and functions; comment text now starts with the salient detail rather than repeating the identifier.
- **`config/config.go`** — re-aligned constant block to `gofmt`-standard single-space alignment.
- **`testing/moxtox`** — renamed receiver `cnfg` → `cfg`; expanded single-line `if !ok { continue }` guards to multi-line form.
- **All test files** — added section-break comments between logical test groups for readability.

---

## [0.10.0]

### Changed

- **`http/client` — circuit breaker** (`http/client/circuitbreaker.go`) — added per-host circuit breaker with configurable failure threshold, timeout, and half-open probe.

---

## [0.9.x and earlier]

Prior releases were not formally versioned. See git history for changes.
