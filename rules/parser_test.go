package rules

import (
	"os"
	"testing"
)

const testYAML = `
rules:
  /users:
    GET:
      level: lenient
      requiredVersion: 1
    POST:
      level: strict
      requiredVersion: 1
      headers:
        Content-Type:
          required: true
          value: "application/json"
      body:
        name:
          type: string
          required: true
        age:
          type: int
  /users/:id:
    GET:
      level: lenient
    PUT:
      level: strict
      requiredVersion: 2
  /items/{itemId}:
    DELETE:
      level: strict
      requiredVersion: 1
`

const invalidYAML = `
rules:
  this is not: [valid: yaml: at all
`

func TestRules_Parser_LoadConfig(t *testing.T) {
	t.Run("no path and no env var returns false", func(t *testing.T) {
		ResetForTesting()
		os.Unsetenv("EVAL_RULES_PATH")
		result := LoadConfig()
		if result {
			t.Error("expected false when no path provided")
		}
	})

	t.Run("with EVAL_RULES_PATH env var", func(t *testing.T) {
		ResetForTesting()
		f, err := os.CreateTemp("", "rules*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		f.WriteString(testYAML)
		f.Close()

		os.Setenv("EVAL_RULES_PATH", f.Name())
		defer os.Unsetenv("EVAL_RULES_PATH")

		result := LoadConfig()
		if !result {
			t.Error("expected true with valid YAML file")
		}
	})

	t.Run("with explicit path", func(t *testing.T) {
		ResetForTesting()
		os.Unsetenv("EVAL_RULES_PATH")
		f, err := os.CreateTemp("", "rules*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		f.WriteString(testYAML)
		f.Close()

		result := LoadConfig(f.Name())
		if !result {
			t.Error("expected true with explicit valid path")
		}
	})

	t.Run("with invalid YAML file", func(t *testing.T) {
		ResetForTesting()
		os.Unsetenv("EVAL_RULES_PATH")
		f, err := os.CreateTemp("", "rules*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		f.WriteString(invalidYAML)
		f.Close()

		result := LoadConfig(f.Name())
		if result {
			t.Error("expected false with invalid YAML")
		}
	})

	t.Run("with nonexistent path", func(t *testing.T) {
		ResetForTesting()
		os.Unsetenv("EVAL_RULES_PATH")
		result := LoadConfig("/nonexistent/path/rules.yaml")
		if result {
			t.Error("expected false for nonexistent file")
		}
	})

	t.Run("retry on failure succeeds", func(t *testing.T) {
		ResetForTesting()
		os.Unsetenv("EVAL_RULES_PATH")

		result := LoadConfig("/nonexistent/path/rules.yaml")
		if result {
			t.Error("expected false for nonexistent file")
		}

		f, err := os.CreateTemp("", "rules*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		f.WriteString(testYAML)
		f.Close()

		result = LoadConfig(f.Name())
		if !result {
			t.Error("expected true on retry with valid file")
		}
	})
}

func TestRules_Parser_LoadConfigWithData(t *testing.T) {
	t.Run("valid YAML", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(testYAML))
		if !IsConfigLoaded() {
			t.Error("expected config to be loaded")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(invalidYAML))
		if IsConfigLoaded() {
			t.Error("expected config to not be loaded")
		}
	})

	t.Run("YAML with invalid route pattern regex fails", func(t *testing.T) {
		badRegexYAML := `
rules:
  /{[invalid}:
    GET:
      level: lenient
`
		ResetForTesting()
		result := LoadConfigWithData([]byte(badRegexYAML))
		if result {
			t.Error("expected false for invalid route pattern regex")
		}
	})

	t.Run("YAML with invalid field pattern regex fails load", func(t *testing.T) {
		badFieldRegexYAML := `
rules:
  /test:
    GET:
      level: lenient
      headers:
        X-Bad:
          pattern: "^INVALID[[$"
`
		ResetForTesting()
		result := LoadConfigWithData([]byte(badFieldRegexYAML))
		if result {
			t.Error("expected false for invalid field pattern regex")
		}
	})

	t.Run("multiple patterns sorted by specificity", func(t *testing.T) {
		multiPatternYAML := `
rules:
  /a/:id:
    GET:
      level: lenient
  /a/:id/:sub:
    GET:
      level: lenient
  /a/*/extra:
    GET:
      level: lenient
`
		ResetForTesting()
		LoadConfigWithData([]byte(multiPatternYAML))
		if !IsConfigLoaded() {
			t.Error("expected config to be loaded with multiple patterns")
		}
	})

	t.Run("once loaded, second call is no-op", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(testYAML))
		LoadConfigWithData([]byte(`rules: {}`))
		rules := GetRules()
		if _, ok := rules["/users"]; !ok {
			t.Error("expected /users rule to still be present from first load")
		}
	})
}

func TestRules_Parser_GetRule(t *testing.T) {
	ResetForTesting()
	LoadConfigWithData([]byte(testYAML))

	t.Run("known path and method", func(t *testing.T) {
		rule, _ := GetRule("/users", "GET")
		if rule == nil {
			t.Error("expected non-nil rule for /users GET")
		}
		if rule.Level != LevelLenient {
			t.Errorf("Level = %q, want lenient", rule.Level)
		}
	})

	t.Run("known path unknown method", func(t *testing.T) {
		rule, _ := GetRule("/users", "DELETE")
		if rule != nil {
			t.Error("expected nil rule for unknown method")
		}
	})

	t.Run("unknown path", func(t *testing.T) {
		rule, _ := GetRule("/unknown", "GET")
		if rule != nil {
			t.Error("expected nil rule for unknown path")
		}
	})

	t.Run("empty path", func(t *testing.T) {
		rule, _ := GetRule("", "GET")
		if rule != nil {
			t.Error("expected nil rule for empty path")
		}
	})

	t.Run("empty method", func(t *testing.T) {
		rule, _ := GetRule("/users", "")
		if rule != nil {
			t.Error("expected nil rule for empty method")
		}
	})

	t.Run("nil ruleMap (before load)", func(t *testing.T) {
		ResetForTesting()
		rule, _ := GetRule("/users", "GET")
		if rule != nil {
			t.Error("expected nil rule when ruleMap is nil")
		}
	})

	t.Run("pattern route :id param returns params", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(testYAML))
		rule, params := GetRule("/users/123", "GET")
		if rule == nil {
			t.Error("expected non-nil rule for pattern route /users/:id GET")
		}
		if params == nil || params["id"] != "123" {
			t.Errorf("expected params[id]=123, got %v", params)
		}
	})

	t.Run("pattern route {itemId} param returns params", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(testYAML))
		rule, params := GetRule("/items/item-abc", "DELETE")
		if rule == nil {
			t.Error("expected non-nil rule for pattern route /items/{itemId} DELETE")
		}
		if params == nil || params["itemId"] != "item-abc" {
			t.Errorf("expected params[itemId]=item-abc, got %v", params)
		}
	})

	t.Run("versioned path normalized", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(testYAML))
		rule, _ := GetRule("/v1/users", "GET")
		if rule == nil {
			t.Error("expected non-nil rule for /v1/users (normalized to /users)")
		}
	})

	t.Run("exact match returns nil params", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(testYAML))
		rule, params := GetRule("/users", "GET")
		if rule == nil {
			t.Error("expected non-nil rule for exact match")
		}
		if params != nil {
			t.Error("expected nil params for exact match")
		}
	})
}

func TestRules_Parser_IsConfigLoaded(t *testing.T) {
	t.Run("not loaded", func(t *testing.T) {
		ResetForTesting()
		if IsConfigLoaded() {
			t.Error("expected false before loading")
		}
	})

	t.Run("loaded", func(t *testing.T) {
		ResetForTesting()
		LoadConfigWithData([]byte(testYAML))
		if !IsConfigLoaded() {
			t.Error("expected true after loading")
		}
	})
}

func TestRules_Parser_GetRules(t *testing.T) {
	ResetForTesting()
	LoadConfigWithData([]byte(testYAML))
	rules := GetRules()
	if rules == nil {
		t.Error("expected non-nil rules")
	}
	if _, ok := rules["/users"]; !ok {
		t.Error("expected /users in rules")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"/", "/"},
		{"/users", "/users"},
		{"/users/", "/users"},
		{"/v1/users", "/users"},
		{"/v1.2/users", "/users"},
		{"/v1", "/"},
		{"/users?foo=bar", "/users"},
		{"users", "/users"},
		{"?query=only", "/"},
		{"  ?query  ", "/"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizePath(tc.input)
			if got != tc.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestTemplateToRegex(t *testing.T) {
	tests := []struct {
		tpl      string
		wantKeys []string
	}{
		{"/users/:id", []string{"id"}},
		{"/items/{itemId}", []string{"itemId"}},
		{"/a/:b/:c", []string{"b", "c"}},
		{"/wildcard/*", nil},
		{"/plain/path", nil},
		{"/path?query=ignored", nil},
		{"/path/", nil},
	}

	for _, tc := range tests {
		t.Run(tc.tpl, func(t *testing.T) {
			_, keys := templateToRegex(tc.tpl)
			if len(keys) != len(tc.wantKeys) {
				t.Errorf("templateToRegex(%q) keys = %v, want %v", tc.tpl, keys, tc.wantKeys)
			}
		})
	}
}
