# Changelog

All notable changes to the Komodo Forge SDK are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.19.2]

> **`aws/s3` presigned PUT.** Adds `PresignPut` to the S3 client for generating presigned upload URLs — primary use case is direct-to-S3 client uploads (e.g. profile avatar images).

### Added

- **`aws/s3` — `PresignPut` method.** `PresignPut(ctx, bucket, key string, ttl time.Duration, contentType string, contentLength int64) (string, error)` generates a presigned S3 PUT URL. `ttl` must be greater than zero. `contentType` is included in the signed request only when non-empty. `contentLength`, when greater than zero, is included in the signature — S3 enforces an exact byte match on upload (useful for fixed-size payloads). Added to the `API` interface.

---

## [0.19.1]

> **Per-package constants.** Well-known env-var keys move out of the central `constants`/`config` package into folder-level `constants.go` files — each in `package <folder>`, so a package owns its own keys (`aws.AWS_REGION`, `http.HOST`, `security.JWT_ISSUER`, …). The `aws` package additionally carries deployment-sizing value constants (regions, ARN partitions, Fargate/Lambda CPU/memory/timeout). The central package is removed.

### Added

- **`aws` — `package aws` constants.** Env-var keys `AWS_REGION`, `AWS_ENDPOINT`, `AWS_SECRET_PATH`, plus value constants for regions (`AWS_US_EAST_1`…`AWS_US_WEST_2`), ARN partitions (`ARN_PARTITION_AWS*`), Fargate CPU/memory, Lambda memory/timeout, and generic request timeouts.
- **`http` — env-var keys.** `HOST`, `MAX_CONTENT_LENGTH`, `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`, `BUCKET_TTL_SECOND`, `IP_WHITELIST`, `IP_BLACKLIST`, `IDEMPOTENCY_TTL_SEC`.
- **`security` — env-var keys.** `JWT_PUBLIC_KEY`, `JWT_PRIVATE_KEY`, `JWT_AUDIENCE`, `JWT_ISSUER`, `JWT_KID`.
- **`logging` — env-var keys.** `LOG_LEVEL`.
- **`api` — env-var keys.** `APP_NAME`, `ENV`, `PORT`, `PORT_PRIVATE`, `PORT_METRICS`.

### Removed

- **Central `constants`/`config` package.** `constants/` and `aws/constants/` are deleted; their keys are redistributed to the owning packages above. DynamoDB env-var keys (`DYNAMODB_ENDPOINT`, `DYNAMODB_TABLE`, `DYNAMODB_ACCESS_KEY`, `DYNAMODB_SECRET_KEY`) consolidate into `aws/dynamodb`.

---

## [0.19.0]

> **Security/crypto sweep.** The dead `crypto/` re-export facade is removed, `security/jwt` and `security/oauth` move to the standard `Client`/`New(Config)` pattern (eliminating package-global mutable state and a latent `iss`/`aud` data race), `security/encryption` ships an AES-256-GCM cipher, `security/hashing` adds an Argon2id password hasher, `security/token` adds fail-closed secure-random token generation, and the rest of the `security/` tree is hardened. **Breaking:** `auth.AuthMiddleware` and `api/headers` now take an injected JWT client.

### Changed

- **`security/jwt` — rewritten around `New(ctx, Config) (*Client, error)`.** Signing and verification hang off an immutable `Client` instead of package globals + `InitializeKeys`. `Config` carries `PrivateKeyPEM`, `PublicKeyPEM`, `KID`, `Issuer`, `Audience`, `Leeway`, and an optional `Secrets`/`PrivateKeyName` pair. `New` requires public key + issuer + audience (fail-fast), parses an optional private key (verify-only clients omit it), and rejects a mismatched key pair. Because the key set is now immutable struct state, the prior `iss`/`aud` read-outside-the-lock data race is gone — no mutex, no `atomic`. Verification adds `WithLeeway` (30s default) for clock skew; `ParseClaims` gains the `WithValidMethods(["RS256"])` allowlist for parity with the other verify paths; `SignToken` rejects `ttl <= 0`; and `ExtractTokenFromRequest` matches the Bearer scheme case-insensitively (RFC 6750) and trims surrounding whitespace.
- **`security/oauth` — rewritten around `New(Config) *Validator`.** Scope validation moves off the package-global `allowedScopes` map to an injectable `Validator` (`Config.AllowedScopes`, defaulting to the canonical set when empty); `IsValidScope`/`GetInvalidScopes` are now methods. `IsValidGrantType` stays a stateless free function. The `svc:` prefix bypass is preserved.
- **`security/redaction` — closed a PII leak and tightened key matching.** `RedactString` previously returned any all-digit value unredacted, leaking bare card numbers (PANs); it now redacts numeric strings that are 13–19 digits and Luhn-valid while leaving ordinary numeric IDs alone. `IsSensitiveKey` switched from naive substring matching to token-boundary matching (normalizing `-_. ` separators), so keys like `className` no longer match `ssn` while `set-cookie`/`access_token`/`x-api-key` still do. `RedactValue` now scrubs typed `map[string]string` and `[]string` values in addition to the `any`-typed variants.
- **`security/bannedcustomers` — email is normalized (lowercase + trim) before lookup,** closing a case-variation ban-evasion bypass; `Config.CacheTTL` adds an optional in-memory result cache (default `0` = disabled).
- **`auth.AuthMiddleware` / `api/headers` — migrated to an injected JWT client (breaking).** `AuthMiddleware` is now `AuthMiddleware(*jwt.Client) func(http.Handler) http.Handler`; `api/headers` exposes `SetJWTClient(*jwt.Client)` and validates the `authorization` header through it, replacing the hand-rolled `Bearer` strip.

### Added

- **`security/encryption` — AES-256-GCM `Cipher`.** `New(Config{Key})` (32-byte key, otherwise rejected) returns a `Cipher` with `Encrypt`/`Decrypt`; output is `nonce || ciphertext+tag` with a random per-message nonce. GCM's auth tag provides integrity, so no separate HMAC is needed.
- **`security/jwt` — private-key loading via a `SecretsProvider`.** `Config.Secrets` + `Config.PrivateKeyName` load the signing key from a secrets backend (satisfied by `aws/secretsmanager.Client`) instead of an environment variable.
- **`security/os/host` — `DisableTracing` (`prctl(PR_SET_DUMPABLE, 0)`) and `LockMemory` (`mlockall`),** build-tagged for Linux with no-op fallbacks, alongside the existing `DisableCoreDumps` (now via `golang.org/x/sys/unix`).
- **`security/hashing` — Argon2id password/token hashing.** `New(Config) (*Hasher, error)` returns a `Hasher` with a `Hash(plaintext) (string, error)` method; the package-level `Verify(plaintext, encodedHash) (bool, error)` checks a candidate against a stored hash. `Config` exposes tunable `Memory`/`Time`/`Parallelism`/`SaltLen`/`KeyLen` with OWASP-aligned defaults (64 MiB, t=3, p=2, 16-byte salt, 32-byte key) applied to zero-value fields. Hashes are PHC-encoded (`$argon2id$v=19$m=,t=,p=$salt$hash`, `base64.RawStdEncoding`) so verification reads its parameters from the stored string; a per-hash salt comes from `crypto/rand`, comparison is constant-time (`subtle.ConstantTimeCompare`), and `Verify` rejects non-`argon2id`/non-v19 encodings (downgrade guard). Standardizes password storage for `komodo-user-api`. Promotes `golang.org/x/crypto` to a direct dependency.
- **`security/token` — secure-random token generation.** Stateless, stdlib-only helpers over `crypto/rand`: `Bytes(n) ([]byte, error)`, `Hex(n) (string, error)`, `URLSafe(n) (string, error)` (`base64.RawURLEncoding`, no padding), and `APIKey(prefix) (string, error)` (a `DefaultBytes`=32 / 256-bit URL-safe token, optionally prefixed). Every path is fail-closed — a non-positive length or an RNG read failure returns an error, never a weak fallback. Intended for API keys, opaque/session tokens, password-reset and email-verification tokens, nonces, and OAuth `state`/PKCE verifiers across services; pairs with `security/hashing` for the generate-then-store-the-hash credential pattern.

### Removed

- **`crypto/jwt` and `crypto/oauth`** — the dead re-export facade (zero importers) is deleted.
- **`security/jwt` package globals + `InitializeKeys`** — replaced by the `Client` constructor.

### Security

- Private JWT signing key can now be sourced from a secrets manager rather than a raw `JWT_PRIVATE_KEY` env var.
- Completed the `security/jwt` `iss`/`aud` data-race fix by removing shared mutable state entirely.
- Closed the `security/redaction` bare-PAN logging leak and the `security/bannedcustomers` email-case ban bypass.
- Added `security/hashing` (Argon2id, constant-time verify, downgrade-guarded PHC encoding) so password storage no longer relies on per-service ad-hoc hashing.
- Added `security/token` (fail-closed `crypto/rand` token generation) so token/API-key minting no longer relies on per-service hand-rolled generators, one of which fell back to a predictable timestamp on RNG failure.

---

## [0.18.2]

### Changed

- **`rules` — fixed empty body bypassing required-field checks.** A POST/PUT/PATCH with an empty body now correctly fails when the rule declares `required: true` body fields. Previously returned `true` unconditionally on empty body.
- **`rules` — removed `DisallowUnknownFields` from body JSON decoding.** Unknown fields in the request body are now accepted. Previously, any body field not declared in the YAML config caused a 400 — breaking forward-compatible APIs.
- **`rules` — body fields now enforce `pattern`, `enum`, `min_len`, `max_len` constraints.** These were silently skipped for body fields despite being supported in the YAML schema and enforced on headers, path params, and query params.
- **`rules` — invalid regex in config now fails the entire load.** Previously, a bad `pattern` regex was swallowed as a warning at startup, then silently rejected every matching request forever. Now `LoadConfig` / `LoadConfigWithData` return `false` and the config is not loaded.
- **`rules` — removed dead `Toggle` and `OriginTypes` fields from `EvalRule`.** Neither was checked anywhere.
- **`rules` — extracted common field validation into `validateFieldValue`.** Pattern, enum, min/max length, and type checks are now a single function called by headers, path params, query params, and body validation — eliminating ~120 lines of duplicated logic.
- **`rules` — route matching now happens once per request.** `GetRule` returns both the matched rule and extracted path params, eliminating the redundant second regex pass that `areValidPathParams` previously performed via `matchRouteAndExtractParams`.
- **`rules` — body read/parse skipped when rule has no body constraints.** Avoids unnecessary `io.ReadAll` + JSON decode on POST/PUT/PATCH when the rule's `Body` map is empty.
- **`rules` — enum checks use pre-built `map[string]struct{}` for O(1) lookup.** Enum sets are constructed once at config load instead of linearly scanned per request.
- **`rules` — `LoadConfig` / `LoadConfigWithData` now retry on failure.** Replaced `sync.Once` with `sync.Mutex` + bool guard. A failed load no longer permanently consumes the once-gate; subsequent calls can retry until one succeeds.

### Added

- **`api/validation` — new package for the rule validation middleware.** Moved from `rules/middleware.go`. `Middleware(cfg ...Config)` returns a configurable middleware constructor; `Config.RejectUnmapped` controls whether unmapped routes receive a 400 (default: passthrough). `RuleValidationMiddleware` is a backward-compatible convenience that preserves the old reject-on-no-rule behavior.
- **`api/middleware` — `ValidationMiddleware` export.** Re-exports `api/validation.Middleware` alongside the existing `RuleValidationMiddleware`.

### Removed

- **`rules/middleware.go`** — moved to `api/validation/middleware.go`.

---

## [0.18.1]

### Added

- **`logging/otel` — standalone OpenTelemetry logger.** Instance-based `Logger` struct (not a singleton) with `New(Config) (*Logger, error)` constructor. Built on the official OTEL Go SDK (`go.opentelemetry.io/otel/sdk/log` v0.20.0). Supports both OTLP/HTTP and OTLP/gRPC transports via a `Transport` enum on `Config`, plus simple file output (JSON lines, append mode, no rotation) via a custom `fileExporter` implementing `sdklog.Exporter`. Resource attributes (`service.name`, `service.version`, `deployment.environment`) are auto-attached from `Config` fields with env-var fallbacks (`OTEL_SERVICE_NAME`, `OTEL_SERVICE_VERSION`, `OTEL_DEPLOYMENT_ENVIRONMENT`). Public API: `Trace`, `Debug`, `Info`, `Warn`, `Error`, `Fatal` (severity-level convenience methods), `Publish` (raw `log.Record` emission), `WithAttributes` (child logger with additional base attributes), `Enabled` (level check), `Shutdown`, `ForceFlush`. Level filtering is enforced at the `Logger` layer — methods below the configured severity are no-ops. Attribute helpers (`String`, `Int`, `Int64`, `Float64`, `Bool`, `Bytes`) re-export `log.KeyValue` constructors for ergonomic call-site use.

---

## [0.18.0]

> **Logger rewrite + redaction relocation.** `logging/runtime` is redesigned for async I/O, mandatory redaction-by-default, and remote sink fanout. `Init(Config)` is the sole entry point — new `Config` struct with `Level`, `Format`, `Redact`, and `Sinks` fields. Redaction core relocated from `api/redaction/` to `security/redaction/` for cross-cutting reuse; the HTTP middleware is rewritten to import the shared core, closing a security gap where the middleware and logger used different sensitive-key lists.

### Changed

- **`logging/runtime` — full rewrite around `Init(Config) error`.** `Config` carries four fields: `Level string` (default `"info"`), `Format` (`FormatJSON` zero-value default, `FormatText` for local CLI), `Redact` (`RedactStrict` zero-value default, `RedactKeysOnly`, `RedactOff`), and `Sinks []Sink` for optional remote log destinations. `Init` returns an error when misconfigured (e.g. `FormatText` with remote sinks). The global logger is an `atomic.Pointer[slog.Logger]` — race-free, lock-free reads. All log output goes through a bounded async writer (`asyncWriter`: 4096-slot channel, pooled buffers, single drain goroutine) so callers never block on I/O. `Sync()` flushes the async buffer synchronously; `Close()` drains and tears down all writers. `Fatal` calls `Sync()` before `os.Exit(1)` to guarantee the fatal message is flushed. Stdout is always on (non-negotiable); `Config.Sinks` adds remote destinations additively via `httpSink` (batched to 64KB, flushed every 1s, 3x retry with backoff, drop-on-full so a slow remote never blocks the request path). `FormatText` produces the fixed layout `2025-10-15T12:34:56Z [INFO] request_id | message | key=value`; `FormatJSON` uses `slog.JSONHandler` with `ReplaceAttr` for inline redaction. `LevelFatal = slog.LevelError + 4` is a real custom level, mapped to `"FATAL"` in both formats.

- **`security/redaction` — relocated from `api/redaction/`.** Core redaction functions (`IsSensitiveKey`, `RedactString`, `RedactPair`, `RedactValue`) moved to `security/redaction/` for cross-cutting reuse by both the logger and the HTTP middleware. The sensitive-key list is unified: the middleware's `access_token`, `refresh_token`, `client_secret`, `credit_card`, `creditcard`, `card_number` entries and the core's `x-amz-signature`, `bearer`, `cvv`, `private_key` entries are merged into a single canonical set. Detection uses substring matching instead of exact map lookup, catching variants like `X-Api-Key` and `x_api_key` in one pass.

- **`api/redaction/middleware.go` — rewritten to import shared core.** Duplicate `containsSensitiveKey` (exact-match map), `redactInterface` (deep map/slice redaction), and the per-package sensitive-key list are deleted. Middleware now calls `redact.IsSensitiveKey` and `redact.RedactValue` from `security/redaction/`. HTTP-specific logic (`bearerRE`, `longTokenRE`, `looksLikeToken`, `redactBody`) remains in the middleware. This closes a live security gap where the middleware redacted keys the logger did not, and vice versa.

- **`logging/runtime` — redaction is inline, not a wrapper.** The `redactingHandler` wrapper type is deleted. JSON redaction runs via `slog.HandlerOptions.ReplaceAttr`; text redaction runs inline in the `textHandler.Handle` method. Both call the same `redactAttr(mode, attr)` function, which delegates to `security/redaction` for key detection and value scrubbing.

### Added

- **`logging/runtime` — `Sink` type and remote log shipping.** `Sink{URL string, Headers map[string]string}` in `Config.Sinks` declares a remote log destination (Splunk HEC, New Relic, Grafana Loki, etc.). Each sink runs its own goroutine with batching, retry, and backoff. Newline-delimited JSON is POSTed; custom headers (e.g. `Api-Key`) are forwarded.

- **`logging/runtime` — `Sync()` and `Close()` lifecycle.** `Sync()` flushes the async stdout buffer (blocking). `Close()` drains and tears down the async writer and all remote sinks. Services should `defer logger.Close()` in `main`.

- **`logging/runtime` — `LevelFatal` and `Fatal(msg, err, args...)`.** Custom level at `slog.LevelError + 4`. `Fatal` logs at this level, flushes, and exits. Both text (`[FATAL]`) and JSON (`"level":"FATAL"`) handlers render the label.

- **`logging/runtime` — `RedactMode` enum.** `RedactStrict` (zero value, default): redacts sensitive keys and scans all string values for PII (emails, SSNs, card numbers, Bearer tokens). `RedactKeysOnly`: redacts sensitive keys only, leaves values untouched. `RedactOff`: no redaction. Deep recursive redaction covers groups, `map[string]any`, `[]any`, and string attrs.

### Removed

- **`logging/runtime` — `SetOutput`.** Deleted. Output is configured at `Init` via stdout (always) and `Config.Sinks` (optional remotes). No runtime writer swapping.
- **`logging/runtime` — `Config.Name`, `Config.Version`, `Config.Env`, `Config.Output`, `Config.Handler`.** All removed. `Name`/`Version` are consumer concerns (add as attrs if needed). `Env` is removed entirely — no env-based level floor. `Output` and `Handler` are replaced by the stdout-always + sinks model.
- **`logging/runtime` — ANSI color support.** Deleted. CloudWatch and log aggregators do not interpret ANSI escape codes. Text format is plain.
- **`logging/runtime` — `RedactingLogger` type.** Replaced by inline `redactAttr` called from both handlers. No wrapper handler, no separate logger type.
- **`logging/runtime` — env-based debug-level floor.** The `effectiveLevel` guard that raised `debug` to `info` outside local environments is removed. Level is caller-controlled via `Config.Level` and `SetLevel`.

---

## [0.17.3]

### Added

- **`db/redis` — `(*Client).GetDel` and `(*Client).MGet`.** `GetDel(ctx, key)` is an atomic get-and-delete (Redis `GETDEL`) for single-use-where-read-*is*-redemption — e.g. an `authorization_code` exchange, where the code itself is the secret. `MGet(ctx, keys...)` is a one-round-trip bulk read returning a positional `[]string` with `""` for misses. Both are added as concrete `*Client` methods and deliberately **not** added to the exported `redis.API` interface, so existing interface implementations and mocks across consumers are unaffected (non-breaking). Consumed by `komodo-auth-api`: `GETDEL` for the `authorization_code` exchange, an `MGET` one-RTT saving on OTP-verify. Note: `GetDel` is **not** a substitute for the `SetNX` single-use claim on OTP/passkey redemption — it deletes before any correctness check, so a wrong guess would consume a victim's live code.
---

## [0.17.2]

### Added

- **`api/headers` — `MaxContentLengthMiddleware(maxBytes int64)`.** Enforces a request body-size cap and is re-exported as `middleware.MaxContentLengthMiddleware`. Rejects an oversized declared `Content-Length` with `413 Request Entity Too Large` and wraps the body in `http.MaxBytesReader` for defense-in-depth against an absent or understated `Content-Length` (chunked/streamed bodies). When `maxBytes <= 0` the limit resolves from the `MAX_CONTENT_LENGTH` env var, falling back to the new `headers.DEFAULT_MAX_CONTENT_LENGTH` constant (4096). This closes the enforcement gap left by the existing `content-length` header *rule* check (`isValidContentLength`), which validates the declared header value but neither enforces the body stream nor returns a 413. Lifted out of `komodo-auth-api`, which hand-rolled the same middleware locally.
- **`auth` — `RequireAnyScope(scopes ...string)`.** A general scope guard re-exported as `middleware.RequireAnyScope`: passes the request when the token carries *any* of the listed exact scopes, otherwise returns `403` (`InsufficientScope`) with a `requires <a> or <b>` detail; panics at construction if no scopes are given. Complements `RequireServiceScope` (which matches the `svc:` *prefix* for machine identity) with arbitrary user/permission-scope gating. Lifted out of `komodo-auth-api`, which hand-rolled it as `RequireUserScope`.

### Changed

- **`api/headers` — single-sourced the content-length default.** `isValidContentLength`'s inline `4096` default now references `headers.DEFAULT_MAX_CONTENT_LENGTH`, shared with the new middleware. No behavioral change.

---

## [0.17.1]

### Added

- **`security/os/host` — new package for process-level defense-in-depth.** `DisableCoreDumps()` sets `RLIMIT_CORE` to zero (via `syscall.Setrlimit`) so a crash cannot spill in-memory secrets — notably an RSA signing key — to disk. Build-tagged: the guard applies on the Linux container that ships and is a no-op on non-Linux dev hosts (macOS/Windows), so callers can invoke it unconditionally at startup. Lifted out of `komodo-auth-api`, which previously hand-rolled the same guard locally.

---

## [0.17.0]

> **Auth architecture: central issuance, verify-only SDK.** Token issuance is owned by the Auth API (the sole holder of the private signing key); every other service verifies tokens via the `auth` package and obtains its own service tokens via the OAuth2 `client_credentials` grant. This release adds the keyless service-token client, fences off issuance in the universal SDK surface, and deprecates the `crypto/*` minting shims. Issuance hardening (KMS-backed keys, key rotation, `jti`/revocation) is tracked in `komodo-auth-api`.

### Added

- **`http/client` — `WithServiceAuth` + `ClientCredentialsTokenSource`.** A keyless service-to-service auth primitive: `NewClientCredentialsTokenSource(ServiceAuthConfig)` obtains a token from the Auth API via the OAuth2 `client_credentials` grant (form-encoded `grant_type=client_credentials`), caches it, and proactively refreshes ~15% before expiry; concurrent refreshes collapse to a single upstream fetch via `singleflight`. `WithServiceAuth(base, src) http.RoundTripper` wraps any transport so every outbound request carries `Authorization: Bearer <token>` (the inbound request is never mutated, per the `RoundTripper` contract). Composes into `ClientConfig.Transport` or any generated client's transport. Holds no private signing key — issuance stays with the Auth API. Replaces the previously-planned self-minting `WithServiceAuth` design.

### Changed

- **`security/jwt` — issuance fenced as Auth-API-only.** Package, `InitializeKeys`, and `SignToken` doc comments now state that issuance is intended for the Auth API only (the single service holding the private key); application services must verify via `auth.JWKSVerifier` and obtain service tokens via `http/client.WithServiceAuth`, never minting their own. No behavioral change.
- **`auth.RequireServiceScope` — documented service-identity contract.** Clarified that service identity is conveyed by a `svc:<name>` scope (issued by the Auth API on `client_credentials` tokens, requested via `WithServiceAuth`), with `aud` set to the target service for defense-in-depth (enforced separately by `JWKSVerifier.ExpectedAudience`). No behavioral change.

### Deprecated

- **`crypto/jwt` and `crypto/oauth` re-export shims.** Import `security/jwt` and `security/oauth` directly. The `crypto/jwt` shim in particular exposed token minting (`SignToken`) to every consumer — the canonical paths are `auth` (verify) and `http/client.WithServiceAuth` (obtain). The shims still forward for now; they will be removed in a future release once consumers migrate.

---

## [0.16.3]

### Fixed

- **`security/jwt` — data race on key initialization.** `keysInitialized` was a plain `bool` read outside `keyMutex` by `InitializeKeys`, `SignToken`, `ValidateToken`, `ValidateAndParseClaims`, and `ParseClaims` while `InitializeKeys` wrote it under the write lock — a data race on concurrent first callers (caught by `-race`). It is now an `atomic.Bool`, and `InitializeKeys` re-checks it under the write lock before loading keys. Behavior is unchanged: a failed load (keys absent from the environment) still returns an error and can be retried, rather than being made permanent as a `sync.Once` would.

- **`http/client` — circuit breaker no longer trips on 4xx responses.** The breaker counted any `StatusCode >= 400` as an upstream failure, so client errors (400, 401, 404) opened the breaker and denied subsequent traffic to an otherwise-healthy service. It now counts only `>= 500`. Adds `TestWithCircuitBreaker_DoesNotTripOn4xx` as a regression guard. (The retry path's default `ShouldRetry` already retried only `429`/`5xx`.)

- **`api/request` — `GetClientKey` no longer trusts a spoofable `X-Forwarded-For`.** It returned the leftmost (client-supplied) `X-Forwarded-For` entry unconditionally, letting any caller forge the rate-limit / IP-access-control key. It now ignores the header by default (using the direct peer `RemoteAddr`), and when a trusted-proxy depth is configured via `SetTrustedProxyDepth(n)` or the `TRUSTED_PROXY_DEPTH` env var, reads the client IP `n` entries from the right — the trusted-proxy side — ignoring injected left hops; a header shorter than the configured depth falls back to the peer. Affects both callers (`api/ratelimit`, `api/ipaccess`).

- **`auth/middleware` — internal error detail no longer leaked to clients.** Both `AuthMiddleware` and `Middleware` passed `err.Error()` straight into the client response via `WithDetail` on token extract/validate/verify failures, exposing JWT and crypto internals. All four paths now log the full error internally and return a generic detail (`"missing or malformed authorization header"`, `"token validation failed"`, or `"token verification failed"`).

- **`api/request` — `GetRequestID` falls back to the `X-Request-ID` header.** It read only the request-ID context value (set by `RequestIDMiddleware`) and returned `"unknown"` whenever the middleware had not run (e.g. early panics, error responses on unrouted paths). It now falls back to the inbound `X-Request-ID` header before `"unknown"`, so error envelopes carry the caller's request ID even before middleware runs.

- **`api/ratelimit` — `LoadConfig` no longer clobbered by lazy env defaults.** `loadCfg`'s one-time env loader overwrote a configuration set explicitly via `LoadConfig` on its first call, silently resetting programmatic RPS/burst settings to env defaults. The loader now skips the default load when a config is already present.

### Changed

- **`api/ratelimit` — bucket TTL consolidated onto a single env var.** The distributed-cache TTL (read in `loadEnv`, formerly from `BUCKET_TTL_SECOND`) and the in-memory bucket evictor (`startBucketEvictor`, from `RATE_LIMIT_BUCKET_TTL_SEC`) used two different env var names, so neither was authoritative. Both now read `RATE_LIMIT_BUCKET_TTL_SEC` through `envCfg`; `BUCKET_TTL_SECOND` is no longer consulted. The evictor still defaults to 300s when unset.

- **Comment-standard pass over the 0.16.x diff.** Doc comments across the changed source and test files were brought in line with the SDK comment standard: comments no longer open with the symbol name (they lead with a verb), `BreakerConfig`/`RetryConfig` field docs moved inline, over-long doc blocks were trimmed to the non-obvious contract, and test-function comments that merely restated assertions were removed. No behavioral change.

---

## [0.16.2]

### Added

- **`http/client` — configurable retry-with-backoff via `ClientConfig.Retry`.** `RetryConfig{MaxAttempts, BaseDelay, MaxDelay, ShouldRetry}` is a new opt-in `*Config` field (nil by default, mirroring `CircuitBreaker *BreakerConfig`); when set, `Client.Do` retries a request with exponential backoff (doubling from `BaseDelay`, capped at `MaxDelay`) until `ShouldRetry` rejects the outcome, the attempt budget is exhausted, the request context is done, or the circuit breaker reports `ErrOpen`. The default `ShouldRetry` retries transport errors, `429`, and `5xx` responses. Each retried attempt is replayed via `req.GetBody` (already wired in `PostJSON`) so request bodies survive across attempts, and each attempt is routed through the circuit breaker individually — a breaker that opens mid-retry short-circuits the remaining attempts with `ErrOpen` rather than continuing to hammer the upstream. Adds `BenchmarkClientDo_Retry` establishing happy-path overhead.

---

## [0.16.1]

### Fixed

- **`api/headers` — `Authorization` header validation no longer fails every request.** `ValidateHeaderValue("authorization", …)` was passing the literal `"Bearer <token>"` string to `jwt.ValidateToken`, which expects a bare token; every Bearer-token request failed validation. The `Bearer ` prefix is now stripped (and its presence required) before validation, matching `jwt.ExtractTokenFromRequest`.
- **`api/headers`/`api/csrf` — CSRF token validation now actually validates.** `isValidCSRF` returned `true` for any non-empty header, so CSRF protection was effectively disabled. It now implements the double-submit cookie pattern: `CSRFMiddleware` mints a random token (`csrf.GenerateToken`, `crypto/rand`), sets it as a `csrf_token` cookie (`headers.COOKIE_CSRF_TOKEN`) on every response, and `isValidCSRF` constant-time-compares (`crypto/subtle`) the `X-CSRF-Token` header against that cookie — a forged header alone, without the cookie, is rejected.
- **`http/websocket` — `CheckOrigin` no longer accepts every origin.** The upgrader's `CheckOrigin` returned `true` unconditionally, leaving WebSocket connections open to cross-site hijacking. `SetAllowedOrigins([]string)` now configures an explicit allowlist (case-insensitive exact match against the `Origin` header); requests carrying a non-allowlisted `Origin` are rejected, and the allowlist is empty (deny-all cross-origin) by default — no default-allow. Requests with no `Origin` header (non-browser clients) still pass, matching gorilla's same-origin default.
- **`api/request`/`auth`/`api/csrf`/`api/idempotency` — client type can no longer be forged via an unverified JWT.** `GetClientType` decoded the `Authorization` Bearer payload without checking its signature and used `client_type`/`grant_type`/`scope` claims to grant CSRF and idempotency exemptions — any caller could mint an unsigned-looking token claiming `client_type:"api"` to bypass both. `GetClientType` now checks only `X-API-Key` (falling through to `"browser"`); `AuthMiddleware`/`Middleware` derive `ctxKeys.CLIENT_TYPE_KEY` from cryptographically verified claims (`"api"` when the token carries scopes, `"browser"` otherwise) after `ValidateAndParseClaims`/`Verify`. `CSRFMiddleware` and `IdempotencyMiddleware` now read `CLIENT_TYPE_KEY` from context exclusively — an absent value fails closed as `"browser"` rather than falling back to the forgeable decode.

---

## [0.16.0]

### Added

- **`api/idempotency` — Redis-backed `DistributedCache`, `NewDistributedStore`, and atomic `Store.CheckAndSet`.** `DistributedCache` (built on `db/redis.API`) replaces the placeholder stub, mapping `Store`/`Load`/`Delete`/`StoreIfAbsent` onto `Set`/`Exists`/`Delete`/`SetNX`. `NewDistributedStore(client redis.API, ttl int64) *Store` constructs a `Store` backed by it for multi-instance deployments. `Cache.StoreIfAbsent(key, value, ttl) (bool, error)` is a new race-free check-and-reserve primitive — `LocalCache` implements it via `sync.Map.LoadOrStore` + `CompareAndSwap`, `DistributedCache` via Redis `SETNX` — and `Store.CheckAndSet(key) (bool, error)` exposes it at the store level. `IdempotencyMiddleware` now calls `CheckAndSet` instead of separate `Check`+`Set`, closing a TOCTOU race where two concurrent identical requests could both observe "new" and both proceed. `NewStore(mode, ttl)` remains local-only (constructs `LocalCache`); pass a Redis client through `NewDistributedStore` instead.

- **`security/bannedcustomers` (new package) — `Client`, `Checker`, `Config`, `IsBanned`.** DynamoDB-backed deny-list lookup keyed by email. `IsBanned(ctx, email) (bool, error)` distinguishes "not banned" from "lookup failed" via `errors.Is(err, dynamodb.ErrNotFound)`, proactively treats records whose `expires_at` has passed as not-banned (without waiting on DynamoDB's own TTL sweep), and fails open — logging and reporting "not banned" — on any other lookup error so a DynamoDB outage never blocks legitimate customers. An empty `email` is treated as caller error and returned directly. Replaces the `strings.Contains`-based workaround duplicated in `komodo-auth-api/internal/clients.BannedCustomersClient`.

- **`aws/secretsmanager` — `Client.Watch`.** `Watch(ctx, interval, keys, onChange)` polls the secret blob at `SecretPath` on a supervised background goroutine (panics recovered, loop restarted), re-fetching directly from Secrets Manager each tick — bypassing the read cache so rotations are observed immediately — and invokes `onChange` with the requested keys' current values only when at least one differs from the prior poll. Lets `komodo-auth-api` pick up rotated JWT signing keys without an ECS restart. Because `Close` tears down the client's cache-eviction loop, callers that start a `Watch` must keep the client alive for as long as rotation needs to be observed.

- **`aws/dynamodb` — retry-with-backoff for unprocessed batch items; `dynamoDBAPI` test-injection interface.** `batchWriteItems`/`batchDeleteItems` no longer return a hard error the moment `BatchWriteItem` reports `UnprocessedItems`; `retryUnprocessed` now resends just the unprocessed subset with exponential backoff (50ms base, doubling, capped at 5 attempts), returning a `WrapError`-wrapped terminal error only once retries are exhausted. The package also gained a private `dynamoDBAPI` interface wrapping the raw `*dynamodb.Client` surface (`GetItem`, `PutItem`, `BatchWriteItem`, `Query`, `Scan`, `DescribeTable`, etc.) plus a `newWithAPI` test backdoor, bringing it in line with the SDK's standard test-injection convention and unblocking component tests for batch operations that previously required a live endpoint.

### Changed

- **Release process — version now derived from `CHANGELOG.md` instead of a tracked `VERSION` file.** `make release` reads the most recent `## [x.y.z]` heading from `CHANGELOG.md`, errors out if none is found or the resulting tag already exists, then tags and pushes from that. The `VERSION` file is removed; bumping the release version is now just adding a new heading here. `pre-commit` (renamed `pre-commit.sh`) drops its `VERSION`-bump step accordingly and now only formats and re-stages modified Go files.
- **`TODO.md` is no longer tracked.** Removed from the repo and added to `.gitignore` (`TODO.md`, `**/TODO.md`); it remains a local planning scratchpad rather than a checked-in artifact.

---

## [0.15.6]

### Added

- **`api/handlers/health` — `Checker` interface, `NewReadyHandler`, and built-in checker factories.** `Checker` (`Name() string`, `Check(ctx context.Context) error`) and the `CheckerFunc(name string, fn func(ctx context.Context) error) Checker` adaptor let services register downstream-dependency probes without declaring named types. `NewReadyHandler(checkers []Checker, opts ...Option) http.HandlerFunc` runs every registered checker concurrently, caches each result for `CacheTTL` (default 10s, `WithCacheTTL`) behind a `singleflight`-deduped, mutex-protected map to absorb load-balancer probe spam, and bounds each probe with `CheckTimeout` (default 3s, `WithCheckTimeout`) when the request context carries no deadline. Responds `200 {"status":"OK"}` when every dependency is reachable, or `503 {"failing": [{"dep","error"}]}` with the complete failure list — and the verbatim error per dependency — otherwise; any single failure marks the whole probe unhealthy. The existing static `HealthHandler` is unchanged and remains the liveness probe (`GET /health`); `NewReadyHandler` is wired separately as `GET /health/ready`. Built-in factories — `DynamoDBChecker` (`DescribeTable`), `RedisChecker` (`Ping`), `S3Checker` (`HeadBucket`), `HTTPChecker` (GET with a 2s timeout, checks for 2xx) — cover the common dependency types; services inject whichever they own.

- **`db/redis`, `aws/dynamodb`, `aws/s3` — `Ping`, `DescribeTable`, `HeadBucket` added to `API` and `Client`.** Lightweight reachability probes (`Ping(ctx) error`, `DescribeTable(ctx, table) error`, `HeadBucket(ctx, bucket) error`) added to back the new health-check factories above; existing fakes/stubs implementing these `API` interfaces need the new methods.

---

## [0.15.5]

### Added

- **`db/redis` — `Expire` on `API` interface and `Client`.** `Expire(ctx context.Context, key string, ttl int64) error` sets the TTL on an existing key without touching its value. A `ttl ≤ 0` is a no-op. Needed to fix a TOCTOU race in `komodo-auth-api`'s `IncrOTPAttempts`: the prior pattern called `Incr` then `Set("1", ttl)` to attach a TTL, but `Set` overwrites the value — a concurrent second `Incr` between the two calls reset the counter to 1. `Expire` only touches the TTL, leaving the counter intact. Callers that fake `CacheClientOperations` in tests must add `Expire` to their stubs.

- **`aws/dynamodb` — `ErrNotFound` sentinel and `WrapError` helper.** `ErrNotFound` is a package-level sentinel (`var ErrNotFound = fmt.Errorf("item not found")`) returned by `GetItemAs` when the response item map is empty (item does not exist). `WrapError(err error, operation string) error` maps the full set of typed DynamoDB SDK errors (`ConditionalCheckFailedException`, `ResourceNotFoundException`, `ProvisionedThroughputExceededException`, etc.) to verb-phrase error strings with the operation name embedded; the original error is wrapped so `errors.As` and `errors.Is` still work on the chain. Callers can now use `errors.Is(err, dynamodb.ErrNotFound)` instead of string-matching `err.Error()` for not-found detection.

---

## [0.15.4]

### Changed

- **`aws/secretsmanager` — `Prefix` + `Batch` replaced by a single `SecretPath` field.** `Config.Prefix` and `Config.Batch` are removed; `Config.SecretPath` is the full secret name (e.g. `"komodo-auth-api/local/all-secrets"`). `GetSecrets` no longer takes `prefix`/`batchID` params — it uses `SecretPath` directly. `GetSecret` takes a full `name string` instead of `key + prefix`. Eliminates string concatenation, the trailing-slash ambiguity, and the misnamed `AWS_SECRET_BATCH` env var.
- **`constants` — `AWS_SECRET_PREFIX` and `AWS_SECRET_BATCH` replaced by `AWS_SECRET_PATH`.** Single env var maps to `Config.SecretPath`. Callers set the full path per service per environment; no partial path splitting required.

---

## [0.15.3]

### Changed

- **`logging/runtime` — `Init` signature simplified; `sync.Once` removed.** `version ...string` (variadic) replaced with `version string`; pass `""` to default to `"unknown"`. The variadic form only ever used index 0 and silently swallowed extra arguments, making the API misleading. `sync.Once` was removed alongside it: the package-level vars are already a singleton per binary in Go, and the Once added testing friction (callers had to reach in and reset `initOnce` between tests) without providing any real guarantee that mattered. `Init` is now a plain function — honest, no-ceremony, and trivially testable. The `resetInitOnce` test helper and the idempotency test are removed; `captureInit` replaces the scattered save/restore boilerplate in Init tests.

- **`logging/runtime` + `api/redaction` — logger performance optimizations.** Six targeted improvements across the text handler and redaction layer:

  - **`RedactingLogger.Handle` correctness fix** (`logging/runtime/redaction.go`): `rec.Clone()` was replaced with `slog.NewRecord(...)`. `Clone` copies all attributes into the new record, then the redaction loop added them a second time — every attribute was emitted twice, once unredacted. Now a fresh record is built and only the redacted attrs are added.

  - **Regex → map for sensitive-key lookup** (`api/redaction/redaction.go`): `keyRegex.MatchString(key)` ran a compiled regex on every log attribute key on every record. Replaced with a `map[string]struct{}` looked up via `strings.ToLower(key)` — O(1) hash lookup vs. backtracking regex, ~10-50× faster on the key-check branch.

  - **`sync.Pool` for `bytes.Buffer`** (`logging/runtime/handler.go`): `Handle` was allocating a fresh `bytes.Buffer` per call. A package-level `sync.Pool` now amortizes those allocations under concurrent load, reducing GC pressure.

  - **`[]string` + `strings.Join` → `strings.Builder`** (`logging/runtime/handler.go`): The `parts []string` accumulator and the trailing `strings.Join` produced two extra heap allocations per record. Attrs are now written directly into a `strings.Builder`, eliminating both.

  - **Precomputed `coloredLevel` strings** (`logging/runtime/handler.go`): `coloredLevel` was concatenating ANSI escape codes on every call. The 10 possible outputs (5 levels × plain/color) are now package-level vars computed once at startup; the method returns the appropriate constant.

  - **Removed ownerless TODO comment** (`api/redaction/redaction.go`): stale `// TODO - move common code...` comment removed.

---

## [0.15.2]

### Changed

- **`logging/runtime` — `debug` log level is now floored to `info` outside local environments.** `Init` and `SetLevel` route through a new `effectiveLevel(lvl, env)`: a requested level below `info` is raised to `info` whenever the env is not local (`local`/`dev`/`development`), so `staging`, `production`, and an unset env can never emit `Debug` regardless of the configured `LOG_LEVEL`. This makes "secrets logged at debug for local diagnosis (OTP codes, tokens) never reach non-local logs" a structural guarantee rather than a config-discipline convention. `SetLevel` honors the env captured at `Init`, so a runtime level change cannot bypass the floor. `parseLevel` is unchanged (still pure); local-env behavior and all other levels (`info`/`warn`/`error`) are unaffected.

---

## [0.15.1]

### Changed

- **`testing/testutil` — unit is now the default tier; added a `Component` marker.** Corrects the 0.15.0 ladder: an unset/unrecognized `TEST_TIER` (without `-short`) now resolves to `unit` rather than `component`, so a plain `go test ./...` runs unit tests only. Every non-unit tier is now opt-in via an explicit marker — `Component`, `Integration`, `E2E`, `Chaos` — each calling `t.Helper()` and skipping via `t.Skipf` when the active tier is below it (`"skipping component test: set TEST_TIER=component or higher to run"`). The `unit < component < integration < e2e < chaos` ordering, cumulative gating, and `-short` override are unchanged.

---

## [0.15.0]

### Added

- **`testing/testutil` — universal test-tier gating helpers (`package testutil`).** New `testing/testutil/tiers.go` implementing the org-wide test-tier ladder from `testing-go.md` §1: an ordered, cumulative ladder `unit < component < integration < e2e < chaos` selected by a single `TEST_TIER` env var. The active tier is the highest enabled tier; a test of tier `T` runs iff `active >= T`. Selection rules: `go test -short` overrides everything and forces unit-only; an unset/unrecognized `TEST_TIER` (without `-short`) defaults to `component`. Exposes exactly three skip-helpers — `Integration`, `E2E`, `Chaos` — each takes `*testing.T`, calls `t.Helper()`, and skips via `t.Skipf` when the active tier is below the helper's tier, naming the tier and how to enable it (`"skipping integration test: set TEST_TIER=integration or higher to run"`). No `Unit`/`Component` helpers: those tiers are always-on. Resolution is centralized in an unexported `resolve(short bool, env string) tier` behind `active()`, table-driven tests colocated. Every Komodo Go service imports these rather than redefining tier gating; supersedes the prior ad-hoc `testing/chaos` and `testing/performance` probes.

---

## [0.14.2]

### Added

- **`auth` — consumer JWT verification via injected `Verifier` (Phase 1).** Implements the local-verify side of the introspect-vs-denylist ADR. Consumers inject `auth.Verifier` rather than calling `POST /v1/oauth/introspect`; Phase 2 (Redis bloom-filter denylist) tracked in TODO.

  - **`auth/verifier.go`** — `Verifier` interface (`Verify(ctx context.Context, token string) (*Claims, error)`); `Claims` struct (Subject, Audience, Scopes, JTI, IsAdmin, IssuedAt, ExpiresAt, Issuer); three sentinel errors (`ErrExpired`, `ErrInvalidSignature`, `ErrInvalidToken`) so callers can branch on failure mode without string matching.

  - **`auth/jwks.go`** — `JWKSVerifier` implementing `Verifier`. Fetches RS256 public keys from a JWKS endpoint (auth-api's `/.well-known/jwks.json`), parses `n`/`e` fields via `encoding/base64` + `math/big.Int`, and caches keys by `kid` (default TTL 5 min). On cache miss the key set is re-fetched once; a second consecutive miss returns `ErrInvalidToken` — no retry loop. Concurrent reads hold only a read lock; the write lock is taken only during cache refresh. `Config` fields: `JWKSURL string`, `CacheTTL time.Duration`, `HTTPClient *http.Client` (defaults: 5 min TTL, 10 s HTTP timeout).

  - **`auth/middleware.go` — `Middleware(v Verifier)`** — new middleware constructor alongside the existing `AuthMiddleware`. Accepts an injected `Verifier` for testability. Maps `ErrExpired` → `Auth.ExpiredToken` (401), all other errors → `Auth.InvalidToken` (401). Populates the same context keys as `AuthMiddleware`: `USER_ID_KEY`, `SESSION_ID_KEY`, `SCOPES_KEY`, `IS_ADMIN_KEY`, `AUTH_VALID_KEY`, `REQUEST_TYPE_KEY`. `AuthMiddleware` deprecated with a doc-comment pointer to the new form.

  - **`auth/verifier_test.go`** — 7 unit tests via `httptest.NewServer` as a mock JWKS endpoint. In-test RS256 keypair generated with `crypto/rsa.GenerateKey`; tokens signed with `golang-jwt/jwt/v5`. Cases: valid token (claims populated), expired token (`ErrExpired`), tampered signature (`ErrInvalidSignature`), stale-cache key rotation (re-fetch succeeds), unknown kid after re-fetch (`ErrInvalidToken`), empty `JWKSURL`, unreachable server.

### Changed

- **`auth/middleware.go` — package declaration corrected.** `package middleware` → `package auth`. Package name now matches the directory; all consumers already imported it as `auth` via the module path — no caller changes required.

- **`api/middleware/exports.go` — alias dropped, `Middleware` exported.** Redundant `mwapiauth` alias removed (package is now self-named `auth`). `Middleware = auth.Middleware` added alongside the existing `AuthMiddleware` and `RequireServiceScope` re-exports.

---

## [0.14.1]

### Added

- **`db/redis` — three new atomic operations on `API` interface and `Client`.**
  - `Incr(ctx context.Context, key string) (int64, error)` — atomically increments the integer value at key by one and returns the new value. Key is created with value `1` if it does not exist (standard Redis `INCR` semantics). Consumed by `komodo-auth-api` to fix a read-increment-write race in `IncrOTPAttempts`.
  - `SetNX(ctx context.Context, key, value string, ttl int64) (bool, error)` — sets key to value only if the key does not already exist; returns `true` if the write occurred, `false` if the key was already present. TTL of 0 sets no expiry. Unblocks atomic OTP-cooldown enforcement, distributed lock patterns, and the `api/idempotency` distributed-cache stub.
  - `Exists(ctx context.Context, key string) (bool, error)` — reports whether a key exists without fetching its value. Aligns `db/redis` with the `gcp/memorystore` API stub contract (`Exists` was already present there). Useful for cache-hit checks on the hot path where the value itself is not needed.

---

## [0.14.0]

### Changed

- **`aws/` vs `db/` split — data-plane clients moved out of `aws/`.** The `aws/` tree was mixing AWS-SDK service wrappers with protocol-native clients (Redis, SQL, OpenSearch) that happen to point at AWS-managed endpoints but speak portable wire protocols. New rule: `aws/X` wraps `aws-sdk-go-v2/service/X` (SigV4, SDK-managed transport); `db/X` wraps a protocol-native client (`pgx`, `go-redis`, `opensearch-go`) with a caller-managed connection pool, portable across AWS/GCP/self-hosted/local. The same logical service can appear in both trees — e.g., `aws/elasticache` (cluster management via SDK) + `db/redis` (RESP data plane). Concrete moves:
  - `aws/aurora` → `db/sql` (package renamed to `sqldb` to avoid shadowing `database/sql`). Aurora-specific wording dropped; client is driver-agnostic.
  - `aws/elasticache` → `db/redis` (package `redis`, go-redis imported as `goredis`). `Config.Endpoint` renamed to `Config.Addr` to match go-redis conventions.
  - `aws/elasticsearch` → `db/opensearch` (package `opensearch`). TODO updated to wire `opensearch-go`.
  - `aws/contactlens` → `aws/connect/contactlens` (Contact Lens is a Connect feature; nested for structure).

- **All AWS client constructors take `ctx context.Context` as the first argument** — breaking change. `New(config Config)` → `New(ctx context.Context, config Config)` across `aws/bedrock`, `aws/cloudwatch/{logs,metrics}`, `aws/connect`, `aws/connect/contactlens`, `aws/dynamodb`, `aws/elasticache`, `aws/lambda`, `aws/opensearch`, `aws/rds`, `aws/s3`, `aws/secretsmanager`, `aws/ses`, `aws/sns`, `aws/sqs`. `ctx` is threaded into `awsconfig.LoadDefaultConfig`, replacing the previous hardcoded `context.Background()`. Callers can now bound startup against IMDS/STS hangs by passing a deadline; passing `context.Background()` preserves the old behaviour. Resolves the 2026-05-12 audit finding for the older clients and applies the same pattern to the new 0.14.0 clients.

- **`api/ratelimit/ratelimiter.go`** — import updated to `db/redis`; `SetElasticacheClient` renamed `SetRedisClient` (leaked-abstraction fix; zero external callers). Internal identifiers also renamed for consistency (`ecHolder` → `redisHolder`, `ecClientVal` → `redisClientVal`, `loadEC` → `loadRedis`).

- **Codebase-wide style pass.** Doc comments on exported symbols collapsed to a single verb-led sentence per `standards/principles.md` across `api/csrf`, `api/headers`, `api/idempotency`, `api/redaction`, `api/request`, `api/sanitization`, `api/server`, `auth/middleware`, `events/{client,event}`, `crypto/jwt`, `http/client/{client,circuitbreaker}`, `security/jwt`. `interface{}` → `any` across `api/redaction`, `api/sanitization`, `security/jwt`. `strings.Split` → `strings.SplitSeq` (Go 1.24 iterator) in `api/sanitization`.

### Added

- **`aws/bedrock` — generative AI inference.** Wraps `aws-sdk-go-v2/service/bedrockruntime`. New `Model` typed string with constants for the supported foundation models (Claude Opus 4.7 / Sonnet 4.6 / Haiku 4.5, Titan Text Express/Lite, Llama 3 70B/8B, Mistral Large). `ParseModel(string) (Model, error)` rejects unknown values with `ErrUnknownModel` — HTTP handlers parse caller-supplied model strings through this gate so the SDK boundary always sees validated values. API: `Invoke` (convenience wrapper that builds Anthropic-format request bodies; non-Anthropic families return a clear error; an empty text response now returns an explicit error instead of `("", nil)`), `InvokeJSON` (raw passthrough), `Converse` (model-agnostic chat). `ConverseStream` deferred — TODO at top of `client.go`. Tests are component-only (interface seam over `bedrockRuntimeAPI`); LocalStack does not support Bedrock.
- **`aws/rds` — RDS Data API.** Full implementation of `aws-sdk-go-v2/service/rdsdata`. API: `ExecuteStatement`, `BatchExecuteStatement`, `BeginTransaction`, `CommitTransaction`, `RollbackTransaction`. `aws/rds/fields.go` provides `toField`/`fromField` converters between Go scalars and the `types.Field` tagged union (`StringValue`, `LongValue`, `DoubleValue`, `BooleanValue`, `BlobValue`, `IsNull`; arrays return an explicit error). Distinct from `db/sql`, which handles wire-protocol Postgres/MySQL via connection pool — `aws/rds` is stateless HTTPS for Lambda and out-of-VPC callers. Tests component-only.
- **`aws/lambda` — function invocation.** Wraps `aws-sdk-go-v2/service/lambda`. API: `Invoke` (sync, returns response payload), `InvokeAsync` (`InvocationType: Event`, fire-and-forget). LocalStack-integration tests.
- **`aws/ses` — transactional email.** Wraps `aws-sdk-go-v2/service/sesv2`. API: `SendEmail` with attachment support via a hand-built `multipart/mixed` MIME message (To/Cc/Bcc/ReplyTo, text + HTML bodies, `Attachment{Filename, ContentType, Data}`). When no attachments, uses the simpler SESv2 `Simple` content type. Component tests via `sesAPI` interface seam plus LocalStack-integration tests.
- **`aws/cloudwatch/metrics` and `aws/cloudwatch/logs` — subpackage split.** Old empty `aws/cloudwatch/client.go` removed. `metrics` wraps `aws-sdk-go-v2/service/cloudwatch` (`PutMetricData` with auto-chunking at 1000 datums/call, `GetMetricStatistics`). `logs` wraps `aws-sdk-go-v2/service/cloudwatchlogs` (`PutLogEvents` with chunking at 10k events or ~1MB, `FilterLogEvents`). LocalStack-integration tests.
- **`aws/connect` — voice contact orchestration.** Wraps `aws-sdk-go-v2/service/connect`. API: `StartOutboundVoiceContact`, `GetContactAttributes`, `UpdateContactAttributes`, `ListContactFlows` (auto-paginated). Tests component-only via interface seam.
- **`aws/connect/contactlens` — call analytics.** Wraps `aws-sdk-go-v2/service/connectcontactlens`. API: `ListRealtimeContactAnalysisSegments` returning flattened `Segment{Type, Content, BeginOffsetMillis, EndOffsetMillis, ParticipantID, Sentiment}`. Tests component-only.
- **`aws/opensearch` — control plane.** Wraps `aws-sdk-go-v2/service/opensearch`. API: `DescribeDomain` (returns flattened `Domain{Name, ARN, Endpoint, EngineVersion, Created, Processing}`), `ListDomainNames`. Separate from `db/opensearch`, which is the REST data plane. LocalStack-integration tests.
- **`aws/elasticache` — control plane.** Wraps `aws-sdk-go-v2/service/elasticache`. API: `DescribeReplicationGroups` (flattened to `{ID, Status, NodeType, NumNodeGroups, Endpoint}`), `DescribeCacheClusters` (`{ID, Status, NodeType, Engine, EngineVersion, NumCacheNodes}`). Separate from `db/redis`, which is the RESP data plane. LocalStack-integration tests.
- **`aws/constants` — US region string constants.** New package exposing `AWS_US_EAST_1`, `AWS_US_EAST_2`, `AWS_US_WEST_1`, `AWS_US_WEST_2`. Callers reference these instead of bare region strings so typos surface as compile errors. No validation gate inside `New` — keeps the constructor surface thin; expand the constant set as additional regions are needed.
- **README — `aws/` vs `db/` rule documented** at the top of the service-packages section.

### Testing

- **LocalStack-only test policy.** AWS SDK packages that LocalStack community supports (Lambda, SES, CloudWatch metrics/logs, OpenSearch control, ElastiCache control, RDS, DynamoDB, S3, SecretsManager, SNS, SQS) ship with integration tests gated by a `net.Dial("tcp","localhost:4566",5s)` probe and `testing.Short()`. Packages LocalStack does not support (Bedrock, Connect, Contact Lens) use component tests via SDK interface mocking. The component-test pattern uses a private interface seam (e.g., `bedrockRuntimeAPI`, `sesAPI`) and a `newWithAPI` test-only constructor.

### Fixed

- **`aws/ses/client.go` — BCC silently dropped on attachment sends.** The raw-MIME branch set only `Content.Raw` and omitted `Destination` on the `SendEmailInput`; because BCC headers are correctly excluded from the MIME envelope, SES had no recipient list and BCC delivery was lost. The branch now also sets `Destination.{To,Cc,Bcc}Addresses`, `FromEmailAddress`, and `ReplyToAddresses` for parity with the Simple-content path. Regression test added.
- **`aws/bedrock/client.go` — `Invoke` silently returning empty string.** When a model response contained no `text`-type content block (e.g., tool-use only), `Invoke` returned `("", nil)`. Now returns `fmt.Errorf("invoke response contained no text content")` so callers can distinguish empty output from a malformed response.
- **`aws/connect/client.go` — placement and formatting.** `API` interface moved to the top of the file (was at the bottom, after the compile-time assertion). `gofmt` drift on struct-literal alignment corrected.
- **`db/redis/client.go` — `NewFromDBString` swallowed parse errors.** `strconv.Atoi` errors silently selected DB 0. Now wrapped and returned. `Config` gains `DialTimeout` and `OpTimeout` (defaults 3s and 2s, preserving prior behaviour); `withTimeout` is a method so per-client overrides apply.
- **`db/redis/client.go` — `interface{}` replaced with `any`** in `AllowDistributed` token-bucket result decoding.
- **`db/sql/client.go` — panic-on-call stubs replaced.** `New` now returns `(nil, ErrNotImplemented)` and `Query`/`Exec` return the same sentinel; callers fail fast at wire time instead of at first query. `ErrNotImplemented` added to `db/sql/errors.go`.
- **`api/csrf/middleware.go` — doc comment weakened to placeholder status.** The previous comment claimed token validation; the underlying `headers.ValidateHeaderValue("X-Csrf-Token", …)` returns `true` unconditionally. Comment now points at the open TODO until real validation lands.
- **Error string convention.** Required-field errors normalised to `"missing X"` form across `aws/bedrock`, `aws/connect`, `aws/connect/contactlens`, `aws/cloudwatch/{logs,metrics}`, `aws/dynamodb`, `aws/elasticache`, `aws/lambda`, `aws/opensearch`, `aws/rds`, `aws/s3`, `aws/secretsmanager`, `aws/ses`, `aws/sns`, `aws/sqs`, `db/redis`. The banned `"invalid input:"` colon prefix removed from `aws/connect`.

---

## [0.13.0]

### Added

- **`gcp/` — GCP service package scaffold (14 packages).** New top-level `gcp/` directory mirroring the `aws/` layout. Each package provides a `Config`, `Client`, `API` interface, and stub methods that return `ErrNotImplemented` until implementation lands. Goal: callers swap providers by changing the import path; method signatures match the AWS counterparts where semantics map cleanly. Packages scaffolded:
  - `gcp/gcs/` — Cloud Storage (parity with `aws/s3`)
  - `gcp/firestore/` — Firestore (parity with `aws/dynamodb`; document-ID model — `BuildKey` intentionally omitted)
  - `gcp/pubsubpub/` — Pub/Sub publisher (parity with `aws/sns`)
  - `gcp/pubsubsub/` — Pub/Sub pull subscriber (parity with `aws/sqs`; no native FIFO ordering keys — divergence documented)
  - `gcp/cloudfunctions/` — Cloud Functions / Cloud Run invoke (parity with `aws/lambda`)
  - `gcp/secretmanager/` — Secret Manager (parity with `aws/secretsmanager`)
  - `gcp/cloudlogging/` — Cloud Logging (parity with CloudWatch logs)
  - `gcp/cloudmonitoring/` — Cloud Monitoring (parity with CloudWatch metrics)
  - `gcp/vertexai/` — Vertex AI generative models (parity with `aws/bedrock`)
  - `gcp/cloudsql/` — Cloud SQL (parity with `aws/aurora`)
  - `gcp/memorystore/` — Memorystore Redis (parity with `aws/elasticache`)
  - `gcp/vertexsearch/` — Vertex AI Search (parity with `aws/elasticsearch`)
  - `gcp/dialogflow/` — Dialogflow CX / CCAI agents (parity with `aws/connect`)
  - `gcp/ccaiinsights/` — Contact Center AI Insights (parity with `aws/contactlens`)

### Changed

- **Performance — per-request regex eliminated.** Five inline `regexp.MustCompile` calls in `api/headers/eval.go` (`isValidUserAgent`, `isValidReferer`, `isValidRequestedBy`, `isValidIdempotencyKey`, `isValidCORS`) and one in `api/redaction/middleware.go` (`sensitiveHeaderRE`) were compiled on every request invocation. All moved to package-level vars. `rules/parser.go` `normalizePath` regex replaced with a 2-byte compare. `api/redaction/redaction.go` adds a fast-path skip for strings shorter than 4 chars or purely numeric.

- **Performance — `api/ratelimit/ratelimiter.go` race fix + single-lock deny path.** `SetElasticacheClient` used an `unsafe.Pointer` atomic store while `Allow` read the field non-atomically — a data race. Replaced with `atomic.Value` + a thin `ecHolder` wrapper; `unsafe` import removed. `allow()` now returns `(bool, time.Duration)` computed while holding the bucket lock, so a denied request no longer re-acquires the lock via a second `retryAfter()` call.

- **Performance — `auth/middleware.go` single JWT parse.** `ValidateToken` + `ParseClaims` parsed the JWT twice per authenticated request. Replaced with a single `jwt.ValidateAndParseClaims` call.

- **Performance — `http/client/client.go` streaming response decode.** `GetJSON` and `PostJSON` replaced `io.ReadAll` + `json.Unmarshal` with `json.NewDecoder(res.Body).Decode` — responses stream without buffering the full body. Error-path bodies are still read to allow connection reuse.

- **Performance — `api/sanitization/middleware.go` redundant regex pass removed.** `sanitizeString` called `SqlInjectionPattern.MatchString(s)` before `ReplaceAllString` — the match check was redundant; removed.

- **Performance — `rules/eval.go` body allocation + log volume.** Body restored via `bytes.NewReader` instead of `bytes.NewBuffer` (avoids an extra copy). Success-path `logger.Info` calls ("all validations passed", "version validation passed", "all headers passed validation") demoted to `logger.Debug` — these fired on every valid request.

- **Performance — `logging/runtime/logger.go` pre-`Init` output suppressed.** Default `slogger` now writes to `io.Discard` until `Init` is called, preventing fully-formatted log output before the service configures a handler.

- **Performance — `api/idempotency/idempotency.go` TTL evictor.** `LocalCache` stored TTL-expiring entries in a `sync.Map` but only deleted them on access — unused expired keys leaked indefinitely. Added a background evictor goroutine (started lazily on first `Store`, runs every minute) that sweeps and deletes expired entries.

- **Performance — `events/client.go` SQS error backoff.** `Subscribe` previously tight-looped on consecutive SQS receive errors with no delay, spinning CPU on misconfigured queues. Replaced with exponential backoff (1s → 30s max, reset on success), respecting context cancellation.

- **Performance — `api/normalization/normalization.go` skip no-op re-encode.** `normalizeQueryParams` parsed and re-encoded `RawQuery` unconditionally. Now skips the `Encode()` write if no key or value was actually changed.

- **Performance — `api/telemetry/middleware.go` structured log fields.** Request telemetry was built into a `map[string]any` then lost via `fmt.Errorf("%v", payload)` string formatting. Replaced with `logger.Attr` calls so fields reach the structured log handler as typed attributes.

- **`api/errors/responses.go` — consistent request ID source.** `RequestId` field was read directly from `req.Header.Get("X-Request-ID")`. Now uses `httpReq.GetRequestID(req)` (reads from context, consistent with the rest of the stack).

---

## [0.12.0]

### Added

- **`codegen/templates/` — oapi-codegen template override.** New top-level `codegen/` package shipping a `client-with-responses.tmpl` that appends a Komodo-standard `New(baseURL string) (*ClientWithResponses, error)` constructor to every generated client. The constructor wires the client to `http/client.NewClient()`, so each consumer service inherits the SDK's HTTP defaults (30s timeout, tuned connection pool, and — when configured — rate limiting + circuit breaker) without hand-writing a wrapper file.
  - **Consumer setup** is two lines in the service's `oapi-codegen.yaml`: a `user-templates` mapping pointing at the SDK template, plus an `additional-imports` entry for `github.com/rdevitto86/komodo-forge-sdk-go/http/client` aliased as `sdkhttp`. After regeneration, calling code reads `comms.New(baseURL)` — no hand-written `client.go` in the service at all.
  - **Deviation** is opt-in: a service that needs custom construction logic drops the `user-templates` line, the generated file falls back to upstream behaviour (no `New` function), and the service hand-writes its own `client.go`. `grep -L user-templates apis/*/internal/clients/*/oapi-codegen.yaml` finds anyone deviating.
  - **Template maintenance:** the override is a verbatim copy of upstream's `client-with-responses.tmpl` plus a clearly-delimited `─── Komodo additions ───` block at the bottom. When oapi-codegen ships a major upgrade that touches the template, re-diff and merge — preserve everything below the divider. Upstream is stable across minor versions; expect this maintenance roughly once per major bump.
  - **Tests** in `codegen/templates_test.go` parse the template with stdlib `text/template` (catching syntax errors without pulling oapi-codegen into the SDK dependency graph) and assert the Komodo additions block remains intact. End-to-end generation is exercised by every consumer service's `go generate`.
- **`codegen/README.md`** documents the pattern, the wiring snippet, deviation strategy, and the upstream-resync procedure.

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
