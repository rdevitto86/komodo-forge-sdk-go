# api/adapters

SDK adapters for Komodo internal services. Each `v{N}/<service>/` package is the
canonical Go client for the matching service's `docs/openapi.yaml`. Consumers
import these instead of hand-rolling HTTP calls so version handling, URL
composition, and typed surfaces stay consistent across services.

## Cross-cutting conventions

These apply to every `v{N}/<service>/` client, generated or hand-rolled.

### 1. Per-client API version

Every client constructor takes the API version as an integer:

```go
func NewClient(baseURL string, ver int) (*Client, error)
```

URLs are built as `baseURL + "/v" + ver + path`. The version is captured at
construction; a single process can hold multiple `*Client` values targeting
different versions of the same service (rolling migration, A/B comparison).

Constructors validate:
- `baseURL` is non-empty
- `ver` is in the package-level `supportedVersions` set

Bad input returns `(nil, error)` — never panic, never `nil`-on-bad-input.

### 2. Per-client base URL (not per-call)

Base URL is fixed at construction. Per-call URL override is intentionally
**not** supported. Rationale:

- Matches today's deployment model (one upstream per consumer per env).
- Per-call override invites address-spoofing bugs and complicates auth/mTLS.
- Canary / blue-green routing belongs at the LB or service-mesh layer, not in
  the SDK client.

If a concrete use case appears (e.g. dual-write during migration), revisit —
prefer constructing two clients over adding a per-call knob.

### 3. Standardized client surface

Every client exposes two layers:

1. **Typed methods** — thin, hand-curated wrappers for high-level operations
   (`comms.SendOTP`, `user.GetCredentials`, ...). These hide URL composition,
   payload shape, and any service-specific contracts (template IDs, header
   conventions). Most consumers call only these.

2. **`Raw() *httpc.Client`** — the underlying HTTP client, for routes the
   typed surface does not yet cover. Use sparingly; prefer adding a typed
   method so all consumers benefit.

Adapters do **not** implement retry, timeout, or circuit-breaker logic —
those are `http/client` responsibilities. Adapters stay thin.

### 4. Hand-curated typed-method registry

The typed-methods layer is **deliberately hand-written, not codegen output**.
Codegen (when it lands) emits low-level request/response types and raw HTTP
calls — the typed layer on top is the consumer-friendly surface, added one
method at a time as consuming services need it.

Guidelines:

- Method name describes the operation, not the route (`SendOTP`, not
  `PostSendEmailOTP`).
- Encapsulate service-specific magic (template IDs, header constants, query
  conventions) inside the method so consumers never pass them.
- Wrap errors with the package and method name:
  `fmt.Errorf("communications.SendOTP: %w", err)`.
- Keep request/response structs unexported unless the typed surface exposes
  them as parameters (in which case export them, matching the OpenAPI
  schema names).

## Layout

```
api/adapters/
  README.md              ← this file
  v1/
    auth/
    cart/
    communications/      ← reference implementation
    order/
    order-reservations/
    payments/
    reviews/
    search/
    shop-items/
    support/
    user/
```

When a new major API version ships and the typed surface is verified, add
`v{N+1}/<service>/` alongside the existing version. Both can coexist; consumers
migrate at their own pace by changing the import path.
