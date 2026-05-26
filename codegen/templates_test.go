// Package codegen ships oapi-codegen templates for Komodo services.
//
// This test file lives at the top of the codegen tree (rather than next to the
// .tmpl files) so it can read the templates as filesystem assets. The tests
// validate template syntax with stdlib text/template and assert the Komodo
// additions remain present and correctly shaped.
//
// Full end-to-end verification (running oapi-codegen against a spec) is
// intentionally out of scope: adding github.com/oapi-codegen/oapi-codegen as a
// SDK dependency would pull in chi/gin/echo/iris transitively. Consumer
// services exercise the real generation path on every codegen run.
package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"
)

const templateDir = "templates"

// loadTemplate reads a template file from the templates directory.
func loadTemplate(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(templateDir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

// noopFuncs registers the oapi-codegen template functions referenced in the
// shipped templates as no-ops. text/template requires every called function to
// be registered at parse time, but the actual implementations live in
// oapi-codegen; stub bodies are sufficient to validate the template's syntax.
func noopFuncs() template.FuncMap {
	return template.FuncMap{
		"opts": func() any {
			return struct {
				OutputOptions struct{ ClientTypeName string }
			}{}
		},
		"genParamArgs":               func(any) string { return "" },
		"genParamNames":              func(any) string { return "" },
		"genResponseTypeName":        func(string) string { return "" },
		"genResponsePayload":         func(string) string { return "" },
		"genResponseUnmarshal":       func(any) string { return "" },
		"getResponseTypeDefinitions": func(any) []any { return nil },
		"ucFirst":                    strings.ToUpper,
	}
}

// TestClientWithResponses_ParsesAsTemplate validates the shipped template
// parses without syntax errors. Failures here usually mean an unclosed {{}}
// directive, a malformed range/if, or a typo in a function name.
func TestClientWithResponses_ParsesAsTemplate(t *testing.T) {
	src := loadTemplate(t, "client-with-responses.tmpl")
	if _, err := template.New("client-with-responses").Funcs(noopFuncs()).Parse(src); err != nil {
		t.Fatalf("template.Parse: %v", err)
	}
}

// TestClientWithResponses_KomodoAdditionsBlock asserts the Komodo additions
// section is present, correctly delimited, and contains the expected New()
// signature. This is the contract every consumer service depends on.
func TestClientWithResponses_KomodoAdditionsBlock(t *testing.T) {
	src := loadTemplate(t, "client-with-responses.tmpl")

	wantMarker := "─── Komodo additions ───"
	if !strings.Contains(src, wantMarker) {
		t.Errorf("expected Komodo additions marker %q in template", wantMarker)
	}

	// Match the canonical New() signature. The body can evolve, but the
	// signature is the public contract.
	sigPattern := regexp.MustCompile(`func New\(baseURL string\) \(\*ClientWithResponses, error\) \{`)
	if !sigPattern.MatchString(src) {
		t.Error("expected `func New(baseURL string) (*ClientWithResponses, error) {` in template")
	}

	// Must call NewClientWithResponses with the sdkhttp doer.
	if !strings.Contains(src, "NewClientWithResponses(baseURL, WithHTTPClient(sdkhttp.NewClient()))") {
		t.Error("expected New() body to delegate to NewClientWithResponses + sdkhttp.NewClient")
	}
}

// TestClientWithResponses_UpstreamPreserved guards against accidental deletion
// of upstream template content during merges. Spot-checks a handful of upstream
// markers — if any of these go missing, the template no longer generates a
// functional client.
func TestClientWithResponses_UpstreamPreserved(t *testing.T) {
	src := loadTemplate(t, "client-with-responses.tmpl")

	upstreamMarkers := []string{
		"type ClientWithResponses struct",
		"func NewClientWithResponses(server string, opts ...ClientOption)",
		"func WithBaseURL(baseURL string) ClientOption",
		"type ClientWithResponsesInterface interface",
		"{{range .}}",            // operation loop
		"{{genResponsePayload",   // upstream response-payload helper
		"{{genResponseUnmarshal", // upstream unmarshal helper
	}

	for _, marker := range upstreamMarkers {
		if !strings.Contains(src, marker) {
			t.Errorf("upstream marker missing — template may be corrupted: %q", marker)
		}
	}
}
