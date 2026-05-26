package oauth

import (
	"strings"
)

var allowedScopes = map[string]bool{
	"read":           true,
	"write":          true,
	"admin":          true,
	"checkout:read":  true,
	"checkout:write": true,
	"orders:read":    true,
	"users:profile":  true,
}

// Validates that every space- or comma-separated scope in scope is in the allowed set; svc: prefixed scopes are always valid.
func IsValidScope(scope string) bool {
	if scope == "" {
		return false
	}
	for s := range strings.FieldsSeq(strings.ReplaceAll(scope, ",", " ")) {
		if strings.HasPrefix(s, "svc:") {
			continue
		}
		if !allowedScopes[s] {
			return false
		}
	}
	return true
}

// Returns all unrecognized scopes from the space- or comma-separated scope string.
func GetInvalidScopes(scope string) []string {
	var invalid []string
	for s := range strings.FieldsSeq(strings.ReplaceAll(scope, ",", " ")) {
		if strings.HasPrefix(s, "svc:") {
			continue
		}
		if !allowedScopes[s] {
			invalid = append(invalid, s)
		}
	}
	return invalid
}

// Validates that grantType is one of the recognized OAuth2 grant types.
func IsValidGrantType(grantType string) bool {
	switch grantType {
	case "client_credentials", "authorization_code", "refresh_token":
		return true
	default:
		return false
	}
}
