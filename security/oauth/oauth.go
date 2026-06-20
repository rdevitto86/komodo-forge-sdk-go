package oauth

import (
	"strings"
)

var defaultAllowedScopes = []string{
	"read", "write", "admin",
	"checkout:read", "checkout:write",
	"orders:read", "users:profile",
}

type Config struct {
	AllowedScopes []string
}

type Validator struct {
	allowed map[string]struct{}
}

func New(cfg Config) *Validator {
	scopes := cfg.AllowedScopes
	if len(scopes) == 0 {
		scopes = defaultAllowedScopes
	}
	allowed := make(map[string]struct{}, len(scopes))
	for _, s := range scopes {
		allowed[s] = struct{}{}
	}
	return &Validator{allowed: allowed}
}

func (v *Validator) IsValidScope(scope string) bool {
	if scope == "" {
		return false
	}
	for s := range strings.FieldsSeq(strings.ReplaceAll(scope, ",", " ")) {
		if strings.HasPrefix(s, "svc:") {
			continue
		}
		if _, ok := v.allowed[s]; !ok {
			return false
		}
	}
	return true
}

func (v *Validator) GetInvalidScopes(scope string) []string {
	var invalid []string
	for s := range strings.FieldsSeq(strings.ReplaceAll(scope, ",", " ")) {
		if strings.HasPrefix(s, "svc:") {
			continue
		}
		if _, ok := v.allowed[s]; !ok {
			invalid = append(invalid, s)
		}
	}
	return invalid
}

func IsValidGrantType(grantType string) bool {
	switch grantType {
	case "client_credentials", "authorization_code", "refresh_token":
		return true
	default:
		return false
	}
}
