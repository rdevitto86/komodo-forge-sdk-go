# codegen

OpenAPI code-generation templates shipped by the forge SDK. Currently provides
one override for [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen):
a `client-with-responses.tmpl` that appends a Komodo-standard `New(baseURL)`
constructor to every generated client. The constructor wires the client to
`http/client.NewClient()`, picking up the SDK's transport defaults (timeouts,
connection pool, rate limiting, circuit breaker).

## Why

Without the template override, each consumer service had to hand-write a small
`client.go` with the same 3-line constructor body:

```go
func New(baseURL string, httpClient *http.Client) (*ClientWithResponses, error) {
    if httpClient == nil {
        httpClient = sdkhttp.NewClient().HTTP() // or equivalent
    }
    return NewClientWithResponses(baseURL, WithHTTPClient(httpClient))
}
```

Multiply that by N consumed services per API and M APIs and you have N × M
hand-written files that all say the same thing. The template moves that body
into the codegen path: the function lands in `types.gen.go` for free, and
deviation is opt-in (a service that needs custom logic simply removes the
`user-templates` line from its `oapi-codegen.yaml`).

## How to wire it up

In each consumer service's `internal/clients/<service>/oapi-codegen.yaml`:

```yaml
package: <service>
output: types.gen.go
generate:
  models: true
  client: true
output-options:
  skip-prune: true
  user-templates:
    client-with-responses.tmpl: <relative-path-to-sdk>/codegen/templates/client-with-responses.tmpl
additional-imports:
  - alias: sdkhttp
    package: github.com/rdevitto86/komodo-forge-sdk-go/http/client
```

The `additional-imports` block injects the SDK import into the generated file
without needing to override `imports.tmpl` — upstream already iterates
`.AdditionalImports` in its imports template.

After regeneration, callers do:

```go
import "yourapi/internal/clients/comms"

c, err := comms.New(os.Getenv("COMMS_API_URL"))
```

No `client.go` file. No nil-check boilerplate. The full generated surface
(`ClientWithResponses`, `WithHTTPClient`, etc.) remains available for tests
and specialised transports.

## When to deviate

Drop the `user-templates` line. The generated file falls back to upstream
behaviour (no `New` function). Hand-write `client.go` in the service with
whatever non-standard construction logic is required.

To find who's deviating:

```sh
grep -L user-templates apis/*/internal/clients/*/oapi-codegen.yaml
```

Absence of the line = deviation, explicit and grep-able.

## Maintaining the template across oapi-codegen upgrades

The template is a verbatim copy of upstream
`client-with-responses.tmpl` plus a "Komodo additions" block at the bottom.
When `oapi-codegen` bumps a major version that touches this template, re-diff:

```sh
# Find upstream template for the version you're using
diff codegen/templates/client-with-responses.tmpl \
     "$(go env GOMODCACHE)/github.com/oapi-codegen/oapi-codegen/v2@<ver>/pkg/codegen/templates/client-with-responses.tmpl"
```

Merge upstream changes above the `─── Komodo additions ───` divider; keep
everything below the divider as-is.

oapi-codegen's templates are stable across minor versions — in practice this
maintenance happens at most once per major bump.

## Tests

`templates_test.go` parses each shipped template with stdlib `text/template`
(catching syntax errors without needing the oapi-codegen template engine) and
asserts the Komodo additions block is present and unchanged.
