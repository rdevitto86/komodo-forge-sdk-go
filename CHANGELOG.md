# Changelog

All notable changes to the Komodo Forge SDK are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.10.2]

### Changed

- **Codebase-wide error message cleanup** — removed service-name prefixes (`"sqs: ..."`, `"events: ..."`, etc.) from all `fmt.Errorf` calls and `logger` calls across `aws/sqs`, `aws/sns`, `aws/dynamo`, `aws/elasticache`, `aws/secretsmanager`, `events`, `http/ratelimit`, and `rules`. Messages now read as actions (`"failed to receive message: ..."`) rather than repeating the package name.
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

- **`aws/dynamo` — parallel batch operations** (`aws/dynamo/client.go`, `aws/dynamo/operations.go`)
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
