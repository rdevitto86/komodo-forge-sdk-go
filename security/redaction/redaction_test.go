package redaction

import (
	"testing"
)

func TestRedaction_RedactString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"no PII", "hello world", "hello world"},
		{"email", "contact user@example.com please", "contact [REDACTED] please"},
		{"SSN", "ssn is 123-45-6789", "ssn is [REDACTED]"},
		{"credit card 16 digits", "card 1234567890123456 ok", "card [REDACTED] ok"},
		{"credit card 13 digits", "card 1234567890123 ok", "card [REDACTED] ok"},
		{"bearer token", "token: Bearer abc123def456", "token: [REDACTED]"},
		{"multiple PII", "email user@test.com ssn 987-65-4321", "email [REDACTED] ssn [REDACTED]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactString(tc.input)
			if got != tc.want {
				t.Errorf("RedactString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRedaction_RedactPair(t *testing.T) {
	sensitiveKeys := []string{
		"authorization", "Authorization",
		"set-cookie", "password", "token", "bearer",
		"ssn", "pwd", "secret", "api_key", "cvv", "card_number",
	}

	for _, key := range sensitiveKeys {
		t.Run("sensitive key: "+key, func(t *testing.T) {
			got := RedactPair(key, "some-value")
			if got != "[REDACTED]" {
				t.Errorf("RedactPair(%q, ...) = %v, want [REDACTED]", key, got)
			}
		})
	}

	t.Run("non-sensitive key with string email value", func(t *testing.T) {
		got := RedactPair("x-custom", "email user@example.com")
		if got != "email [REDACTED]" {
			t.Errorf("got %v, want 'email [REDACTED]'", got)
		}
	})

	t.Run("non-sensitive key with plain string value", func(t *testing.T) {
		got := RedactPair("x-custom", "plain text")
		if got != "plain text" {
			t.Errorf("got %v, want 'plain text'", got)
		}
	})

	t.Run("non-sensitive key with empty string value", func(t *testing.T) {
		got := RedactPair("x-custom", "")
		if got != "" {
			t.Errorf("got %v, want empty string", got)
		}
	})

	t.Run("non-sensitive key with int value", func(t *testing.T) {
		got := RedactPair("x-count", 42)
		if got != 42 {
			t.Errorf("got %v, want 42", got)
		}
	})

	t.Run("non-sensitive key with float64 value", func(t *testing.T) {
		got := RedactPair("x-amount", 3.14)
		if got != 3.14 {
			t.Errorf("got %v, want 3.14", got)
		}
	})

	t.Run("non-sensitive key with bool value", func(t *testing.T) {
		got := RedactPair("x-flag", true)
		if got != true {
			t.Errorf("got %v, want true", got)
		}
	})

	t.Run("non-sensitive key with nil value", func(t *testing.T) {
		got := RedactPair("x-nil", nil)
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("non-sensitive key with slice value", func(t *testing.T) {
		val := []any{"a", "b"}
		got := RedactPair("x-list", val)
		if got == nil {
			t.Error("got nil, want slice")
		}
	})

	t.Run("non-sensitive key with map value", func(t *testing.T) {
		val := map[string]any{"k": "v"}
		got := RedactPair("x-map", val)
		if got == nil {
			t.Error("got nil, want map")
		}
	})
}

func TestRedaction_RedactString_BarePAN(t *testing.T) {
	if got := RedactString("4111111111111111"); got != redacted {
		t.Errorf("expected bare Luhn-valid PAN to be redacted, got %q", got)
	}
}

func TestRedaction_RedactString_BareNonPAN_NotRedacted(t *testing.T) {
	cases := []string{
		"4111111111111112",
		"123456789",
		"42",
		"100000",
	}
	for _, in := range cases {
		if got := RedactString(in); got != in {
			t.Errorf("RedactString(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestRedaction_IsSensitiveKey(t *testing.T) {
	sensitive := []string{
		"authorization", "Authorization", "set-cookie", "password",
		"access_token", "x-api-key", "user_ssn", "client_secret",
	}
	for _, k := range sensitive {
		if !IsSensitiveKey(k) {
			t.Errorf("IsSensitiveKey(%q) = false, want true", k)
		}
	}

	notSensitive := []string{"className", "lesson_id", "x-custom", "count", "session_window"}
	for _, k := range notSensitive {
		if IsSensitiveKey(k) {
			t.Errorf("IsSensitiveKey(%q) = true, want false (over-match)", k)
		}
	}
}

func TestRedaction_RedactValue_TypedMap(t *testing.T) {
	in := map[string]string{"password": "hunter2", "name": "alice"}
	out, ok := RedactValue(in).(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", RedactValue(in))
	}
	if out["password"] != redacted {
		t.Errorf("password = %q, want redacted", out["password"])
	}
	if out["name"] != "alice" {
		t.Errorf("name = %q, want alice", out["name"])
	}
}

func TestRedaction_RedactValue_TypedSlice(t *testing.T) {
	out, ok := RedactValue([]string{"contact user@example.com", "plain"}).([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", RedactValue([]string{}))
	}
	if out[0] != "contact [REDACTED]" {
		t.Errorf("got %q, want redacted email", out[0])
	}
	if out[1] != "plain" {
		t.Errorf("got %q, want plain", out[1])
	}
}
