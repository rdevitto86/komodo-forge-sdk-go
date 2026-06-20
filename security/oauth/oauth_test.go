package oauth

import (
	"testing"
)

func defaultValidator() *Validator { return New(Config{}) }

func TestOAuth_IsValidScope_SingleValid_Success(t *testing.T) {
	v := defaultValidator()
	validScopes := []string{
		"read", "write", "admin",
		"checkout:read", "checkout:write",
		"orders:read", "users:profile",
	}
	for _, s := range validScopes {
		if !v.IsValidScope(s) {
			t.Errorf("IsValidScope(%q) = false, want true", s)
		}
	}
}

func TestOAuth_IsValidScope_MultipleSpaceSeparated_Success(t *testing.T) {
	if !defaultValidator().IsValidScope("read write admin") {
		t.Error("expected true for 'read write admin'")
	}
}

func TestOAuth_IsValidScope_CommaSeparated_Success(t *testing.T) {
	if !defaultValidator().IsValidScope("read,write") {
		t.Error("expected true for comma-separated valid scopes")
	}
}

func TestOAuth_IsValidScope_SvcPrefix_Success(t *testing.T) {
	if !defaultValidator().IsValidScope("svc:internal") {
		t.Error("expected true for svc: prefixed scope")
	}
}

func TestOAuth_IsValidScope_SvcMixedWithValid_Success(t *testing.T) {
	if !defaultValidator().IsValidScope("read svc:internal") {
		t.Error("expected true for svc: mixed with valid scope")
	}
}

func TestOAuth_IsValidScope_Empty_Failure(t *testing.T) {
	if defaultValidator().IsValidScope("") {
		t.Error("expected false for empty scope")
	}
}

func TestOAuth_IsValidScope_Invalid_Failure(t *testing.T) {
	if defaultValidator().IsValidScope("invalid_scope") {
		t.Error("expected false for unknown scope")
	}
}

func TestOAuth_IsValidScope_MixedValidAndInvalid_Failure(t *testing.T) {
	if defaultValidator().IsValidScope("read bad_scope") {
		t.Error("expected false when scope contains an invalid entry")
	}
}

func TestOAuth_New_CustomAllowedScopes(t *testing.T) {
	v := New(Config{AllowedScopes: []string{"custom:read", "custom:write"}})
	if !v.IsValidScope("custom:read") {
		t.Error("expected custom scope to be valid")
	}
	if v.IsValidScope("read") {
		t.Error("expected default scope to be invalid when a custom set is configured")
	}
}

func TestOAuth_GetInvalidScopes_AllValid_Success(t *testing.T) {
	got := defaultValidator().GetInvalidScopes("read write")
	if len(got) != 0 {
		t.Errorf("expected no invalid scopes, got %v", got)
	}
}

func TestOAuth_GetInvalidScopes_SvcIgnored_Success(t *testing.T) {
	got := defaultValidator().GetInvalidScopes("svc:internal svc:other")
	if len(got) != 0 {
		t.Errorf("expected no invalid scopes for svc: scopes, got %v", got)
	}
}

func TestOAuth_GetInvalidScopes_Empty_Success(t *testing.T) {
	got := defaultValidator().GetInvalidScopes("")
	if len(got) != 0 {
		t.Errorf("expected empty slice for empty scope, got %v", got)
	}
}

func TestOAuth_GetInvalidScopes_HasInvalid_Failure(t *testing.T) {
	got := defaultValidator().GetInvalidScopes("read bad_scope another_bad")
	if len(got) != 2 {
		t.Errorf("expected 2 invalid scopes, got %v (len=%d)", got, len(got))
	}
	found := map[string]bool{}
	for _, s := range got {
		found[s] = true
	}
	if !found["bad_scope"] || !found["another_bad"] {
		t.Errorf("unexpected invalid scopes: %v", got)
	}
}

func TestOAuth_IsValidGrantType_Valid_Success(t *testing.T) {
	for _, gt := range []string{"client_credentials", "authorization_code", "refresh_token"} {
		if !IsValidGrantType(gt) {
			t.Errorf("IsValidGrantType(%q) = false, want true", gt)
		}
	}
}

func TestOAuth_IsValidGrantType_Invalid_Failure(t *testing.T) {
	for _, gt := range []string{"", "implicit", "password", "device_code", "unknown"} {
		if IsValidGrantType(gt) {
			t.Errorf("IsValidGrantType(%q) = true, want false", gt)
		}
	}
}
