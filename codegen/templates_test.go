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

func loadTemplate(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(templateDir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

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

func TestClientWithResponses_ParsesAsTemplate(t *testing.T) {
	src := loadTemplate(t, "client-with-responses.tmpl")
	if _, err := template.New("client-with-responses").Funcs(noopFuncs()).Parse(src); err != nil {
		t.Fatalf("template.Parse: %v", err)
	}
}

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
