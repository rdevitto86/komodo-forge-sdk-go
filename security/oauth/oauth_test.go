package oauth

import (
	"testing"
)

func TestOAuth_IsValidScope_SingleValid_Success(t *testing.T) {
	validScopes := []string{
		"read", "write", "admin",
		"checkout:read", "checkout:write",
		"orders:read", "users:profile",
	}
	for _, s := range validScopes {
		if !IsValidScope(s) {
			t.Errorf("IsValidScope(%q) = false, want true", s)
		}
	}
}

func TestOAuth_IsValidScope_MultipleSpaceSeparated_Success(t *testing.T) {
	if !IsValidScope("read write admin") {
		t.Error("expected true for 'read write admin'")
	}
}

func TestOAuth_IsValidScope_CommaSeparated_Success(t *testing.T) {
	if !IsValidScope("read,write") {
		t.Error("expected true for comma-separated valid scopes")
	}
}

func TestOAuth_IsValidScope_SvcPrefix_Success(t *testing.T) {
	if !IsValidScope("svc:internal") {
		t.Error("expected true for svc: prefixed scope")
	}
}

func TestOAuth_IsValidScope_SvcMixedWithValid_Success(t *testing.T) {
	if !IsValidScope("read svc:internal") {
		t.Error("expected true for svc: mixed with valid scope")
	}
}

func TestOAuth_IsValidScope_Empty_Failure(t *testing.T) {
	if IsValidScope("") {
		t.Error("expected false for empty scope")
	}
}

func TestOAuth_IsValidScope_Invalid_Failure(t *testing.T) {
	if IsValidScope("invalid_scope") {
		t.Error("expected false for unknown scope")
	}
}

func TestOAuth_IsValidScope_MixedValidAndInvalid_Failure(t *testing.T) {
	if IsValidScope("read bad_scope") {
		t.Error("expected false when scope contains an invalid entry")
	}
}

func TestOAuth_GetInvalidScopes_AllValid_Success(t *testing.T) {
	got := GetInvalidScopes("read write")
	if len(got) != 0 {
		t.Errorf("expected no invalid scopes for 'read write', got %v", got)
	}
}

func TestOAuth_GetInvalidScopes_SvcIgnored_Success(t *testing.T) {
	got := GetInvalidScopes("svc:internal svc:other")
	if len(got) != 0 {
		t.Errorf("expected no invalid scopes for svc: scopes, got %v", got)
	}
}

func TestOAuth_GetInvalidScopes_Empty_Success(t *testing.T) {
	got := GetInvalidScopes("")
	if len(got) != 0 {
		t.Errorf("expected empty slice for empty scope, got %v", got)
	}
}

func TestOAuth_GetInvalidScopes_HasInvalid_Failure(t *testing.T) {
	got := GetInvalidScopes("read bad_scope another_bad")
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

func TestOAuth_GetInvalidScopes_AllInvalid_Failure(t *testing.T) {
	got := GetInvalidScopes("bad1 bad2")
	if len(got) != 2 {
		t.Errorf("expected 2 invalid scopes, got %v", got)
	}
}

func TestOAuth_IsValidGrantType_ClientCredentials_Success(t *testing.T) {
	if !IsValidGrantType("client_credentials") {
		t.Error("expected true for client_credentials")
	}
}

func TestOAuth_IsValidGrantType_AuthorizationCode_Success(t *testing.T) {
	if !IsValidGrantType("authorization_code") {
		t.Error("expected true for authorization_code")
	}
}

func TestOAuth_IsValidGrantType_RefreshToken_Success(t *testing.T) {
	if !IsValidGrantType("refresh_token") {
		t.Error("expected true for refresh_token")
	}
}

func TestOAuth_IsValidGrantType_Empty_Failure(t *testing.T) {
	if IsValidGrantType("") {
		t.Error("expected false for empty grant type")
	}
}

func TestOAuth_IsValidGrantType_Invalid_Failure(t *testing.T) {
	invalidTypes := []string{"implicit", "password", "device_code", "unknown"}
	for _, gt := range invalidTypes {
		if IsValidGrantType(gt) {
			t.Errorf("IsValidGrantType(%q) = true, want false", gt)
		}
	}
}
