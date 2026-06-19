package redaction

import (
	"regexp"
	"strings"
)

const redacted = "[REDACTED]"

var (
	sensitiveKeys = []string{
		"authorization", "cookie", "set-cookie",
		"password", "passwd", "pwd",
		"token", "access_token", "refresh_token", "id_token",
		"secret", "client_secret",
		"api_key", "apikey", "x-api-key",
		"bearer", "ssn", "cvv",
		"card_number", "credit_card", "creditcard",
		"private_key", "x-amz-signature",
	}
	piiRegex = regexp.MustCompile(`(?i)` +
		`([a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,})|` +
		`(\b\d{3}-\d{2}-\d{4}\b)|` +
		`(\b(?:\d[ -]*?){13,16}\b)|` +
		`(Bearer\s+[a-zA-Z0-9\-\._~\+\/]+=*)`,
	)
)

func RedactString(val string) string {
	if len(val) < 4 {
		return val
	}
	if isNumeric(val) {
		return val
	}
	return piiRegex.ReplaceAllString(val, redacted)
}

func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if k == s || strings.Contains(k, s) {
			return true
		}
	}
	return false
}

func RedactPair(key string, val any) any {
	if IsSensitiveKey(key) {
		return redacted
	}
	return RedactValue(val)
}

func RedactValue(val any) any {
	switch t := val.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, v := range t {
			if IsSensitiveKey(k) {
				out[k] = redacted
				continue
			}
			out[k] = RedactValue(v)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = RedactValue(t[i])
		}
		return out
	case string:
		return RedactString(t)
	default:
		return val
	}
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
