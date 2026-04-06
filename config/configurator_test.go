package config

import (
	"os"
	"testing"
)

// --- GetConfigValue ---

func TestConfigurator_GetConfigValue_FromMemory_Success(t *testing.T) {
	SetConfigValue("CFG_TEST_MEM", "mem-value")
	defer DeleteConfigValue("CFG_TEST_MEM")

	if got := GetConfigValue("CFG_TEST_MEM"); got != "mem-value" {
		t.Errorf("GetConfigValue = %q, want mem-value", got)
	}
}

func TestConfigurator_GetConfigValue_FallbackToEnv_Success(t *testing.T) {
	os.Setenv("CFG_TEST_ENV_FALLBACK", "env-value")
	defer os.Unsetenv("CFG_TEST_ENV_FALLBACK")
	DeleteConfigValue("CFG_TEST_ENV_FALLBACK")

	if got := GetConfigValue("CFG_TEST_ENV_FALLBACK"); got != "env-value" {
		t.Errorf("GetConfigValue fallback = %q, want env-value", got)
	}
}

func TestConfigurator_GetConfigValue_EmptyKey_Failure(t *testing.T) {
	if got := GetConfigValue(""); got != "" {
		t.Errorf("GetConfigValue(\"\") = %q, want empty", got)
	}
}

func TestConfigurator_GetConfigValue_MissingKey_Failure(t *testing.T) {
	os.Unsetenv("CFG_TEST_MISSING_XYZ")
	DeleteConfigValue("CFG_TEST_MISSING_XYZ")

	if got := GetConfigValue("CFG_TEST_MISSING_XYZ"); got != "" {
		t.Errorf("GetConfigValue missing key = %q, want empty", got)
	}
}

func TestConfigurator_GetConfigValue_NilInstance_Failure(t *testing.T) {
	old := instance
	instance = nil
	defer func() { instance = old }()

	if got := GetConfigValue("any"); got != "" {
		t.Errorf("expected empty string with nil instance, got %q", got)
	}
}

// --- SetConfigValue ---

func TestConfigurator_SetConfigValue_Success(t *testing.T) {
	SetConfigValue("CFG_TEST_SET", "hello")
	defer DeleteConfigValue("CFG_TEST_SET")

	if got := GetConfigValue("CFG_TEST_SET"); got != "hello" {
		t.Errorf("after Set, GetConfigValue = %q, want hello", got)
	}
}

func TestConfigurator_SetConfigValue_EmptyKey_Failure(t *testing.T) {
	// Empty key is a no-op — must not panic.
	SetConfigValue("", "value")
}

func TestConfigurator_SetConfigValue_EmptyValue_Failure(t *testing.T) {
	// Empty value is a no-op; key must not be stored.
	SetConfigValue("CFG_TEST_EMPTY_VAL", "")
	if got := GetConfigValue("CFG_TEST_EMPTY_VAL"); got != "" {
		t.Errorf("expected no-op for empty value, got %q", got)
	}
}

func TestConfigurator_SetConfigValue_NilInstance_Failure(t *testing.T) {
	old := instance
	instance = nil
	defer func() { instance = old }()
	// Must not panic with nil instance.
	SetConfigValue("KEY", "val")
}

// --- DeleteConfigValue ---

func TestConfigurator_DeleteConfigValue_Success(t *testing.T) {
	SetConfigValue("CFG_TEST_DEL", "to-delete")
	DeleteConfigValue("CFG_TEST_DEL")

	if got := GetConfigValue("CFG_TEST_DEL"); got != "" {
		t.Errorf("after DeleteConfigValue, GetConfigValue = %q, want empty", got)
	}
}

func TestConfigurator_DeleteConfigValue_EmptyKey_Failure(t *testing.T) {
	// Must not panic with empty key.
	DeleteConfigValue("")
}

func TestConfigurator_DeleteConfigValue_NilInstance_Failure(t *testing.T) {
	old := instance
	instance = nil
	defer func() { instance = old }()
	DeleteConfigValue("KEY")
}

// --- GetAllKeys ---

func TestConfigurator_GetAllKeys_Success(t *testing.T) {
	SetConfigValue("CFG_TEST_ALLKEYS_A", "1")
	SetConfigValue("CFG_TEST_ALLKEYS_B", "2")
	defer func() {
		DeleteConfigValue("CFG_TEST_ALLKEYS_A")
		DeleteConfigValue("CFG_TEST_ALLKEYS_B")
	}()

	keys := GetAllKeys()
	found := make(map[string]bool, len(keys))
	for _, k := range keys {
		found[k] = true
	}
	if !found["CFG_TEST_ALLKEYS_A"] || !found["CFG_TEST_ALLKEYS_B"] {
		t.Errorf("GetAllKeys missing expected keys, got %v", keys)
	}
}

func TestConfigurator_GetAllKeys_NilInstance_Failure(t *testing.T) {
	old := instance
	instance = nil
	defer func() { instance = old }()

	keys := GetAllKeys()
	if len(keys) != 0 {
		t.Errorf("expected empty slice with nil instance, got %v", keys)
	}
}
