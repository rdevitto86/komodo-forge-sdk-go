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
	keyDelimiters           = regexp.MustCompile(`[-_. ]+`)
	normalizedSensitiveKeys = normalizeKeys(sensitiveKeys)
	piiRegex                = regexp.MustCompile(`(?i)` +
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
		if looksLikePAN(val) {
			return redacted
		}
		return val
	}
	return piiRegex.ReplaceAllString(val, redacted)
}

func IsSensitiveKey(key string) bool {
	nk := "_" + normalizeKey(key) + "_"
	for _, s := range normalizedSensitiveKeys {
		if strings.Contains(nk, "_"+s+"_") {
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
	case map[string]string:
		out := make(map[string]string, len(t))
		for k, v := range t {
			if IsSensitiveKey(k) {
				out[k] = redacted
				continue
			}
			out[k] = RedactString(v)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = RedactValue(t[i])
		}
		return out
	case []string:
		out := make([]string, len(t))
		for i := range t {
			out[i] = RedactString(t[i])
		}
		return out
	case string:
		return RedactString(t)
	default:
		return val
	}
}

func normalizeKey(key string) string {
	return keyDelimiters.ReplaceAllString(strings.ToLower(key), "_")
}

func normalizeKeys(keys []string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = normalizeKey(k)
	}
	return out
}

func looksLikePAN(s string) bool {
	if len(s) < 13 || len(s) > 19 {
		return false
	}
	return luhnValid(s)
}

func luhnValid(s string) bool {
	sum := 0
	double := false
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
