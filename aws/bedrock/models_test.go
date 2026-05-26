package bedrock

import (
	"errors"
	"testing"
)

// ── Model enum tests ──────────────────────────────────────────────────────────

func TestParseModel_Valid(t *testing.T) {
	cases := []Model{
		ModelClaudeOpus4_7,
		ModelClaudeSonnet4_6,
		ModelClaudeHaiku4_5,
		ModelTitanTextExpress,
		ModelTitanTextLite,
		ModelLlama3_70B,
		ModelLlama3_8B,
		ModelMistralLarge,
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
		"anthropic.claude-opus-4-7",      // incomplete — missing version suffix
		"anthropic.claude-sonnet-4-6-v1", // missing ":0" suffix
		"anthropic.claude-haiku-4-5-20251001-v1:1", // wrong version number
		"amazon.titan-text-express",                // missing "-v1"
		"meta.llama3-8b-instruct-v1",               // missing ":0"
		"mistral.mistral-large",                    // missing version
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
	if !ModelClaudeSonnet4_6.IsValid() {
		t.Errorf("IsValid() = false for known model %q", ModelClaudeSonnet4_6)
	}
	if Model("bogus").IsValid() {
		t.Errorf("IsValid() = true for unknown model %q", "bogus")
	}
}

func TestModel_String(t *testing.T) {
	want := "anthropic.claude-sonnet-4-6-v1:0"
	if ModelClaudeSonnet4_6.String() != want {
		t.Errorf("String() = %q, want %q", ModelClaudeSonnet4_6.String(), want)
	}
}
