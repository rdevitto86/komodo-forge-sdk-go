package bedrock

import (
	"errors"
	"testing"
)

// ── Unit Tests ───────────────────────────────────────────────────────────────

func TestParseModel_Valid(t *testing.T) {
	cases := []Model{
		MODEL_CLAUDE_DEFAULT,
		MODEL_CLAUDE_HAIKU,
		MODEL_GEMINI_DEFAULT,
		MODEL_GPT_DEFAULT,
		MODEL_DEEPSEEK_DEFAULT,
	}
	for _, m := range cases {
		t.Run(string(m), func(t *testing.T) {
			got, err := ParseModel(string(m))
			if err != nil {
				t.Fatalf("ParseModel(%q) returned unexpected error: %v", m, err)
			}
			if got != m {
				t.Fatalf("ParseModel(%q) = %q, want %q", m, got, m)
			}
		})
	}
}

func TestParseModel_Invalid(t *testing.T) {
	cases := []string{
		"",
		"garbage",
		"anthropic.claude-sonnet-4-6",
		"google.gemini-2.5-pro",
		"openai.gpt-oss-20b",
	}
	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			_, err := ParseModel(s)
			if err == nil {
				t.Fatalf("ParseModel(%q) expected error, got nil", s)
			}
			if !errors.Is(err, ErrUnknownModel) {
				t.Fatalf("ParseModel(%q) error = %v, want ErrUnknownModel", s, err)
			}
		})
	}
}

func TestModels_Deterministic(t *testing.T) {
	a := Models()
	b := Models()
	if len(a) != len(b) {
		t.Fatalf("Models() returned differing lengths: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Models() not deterministic: index %d differs (%q vs %q)", i, a[i], b[i])
		}
	}
	if len(a) != len(validModels) {
		t.Fatalf("Models() returned %d entries, want %d", len(a), len(validModels))
	}
}

func TestModel_IsValid(t *testing.T) {
	if !MODEL_CLAUDE_DEFAULT.IsValid() {
		t.Errorf("IsValid() = false for known model %q", MODEL_CLAUDE_DEFAULT)
	}
	if Model("bogus").IsValid() {
		t.Errorf("IsValid() = true for unknown model %q", "bogus")
	}
}

func TestModel_String(t *testing.T) {
	want := "anthropic.claude-sonnet-4-6-v1:0"
	if MODEL_CLAUDE_DEFAULT.String() != want {
		t.Errorf("String() = %q, want %q", MODEL_CLAUDE_DEFAULT.String(), want)
	}
}
