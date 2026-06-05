package redaction

import (
	"regexp"
	"strings"
)

var (
	sensitiveKeys = map[string]struct{}{
		"authorization": {}, "set-cookie": {}, "password": {}, "token": {},
		"bearer": {}, "ssn": {}, "pwd": {}, "secret": {},
		"api_key": {}, "cvv": {}, "card_number": {},
	}
	piiRegex = regexp.MustCompile(`(?i)` +
		`([a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,})|` + // Email
		`(\b\d{3}-\d{2}-\d{4}\b)|` + // SSN
		`(\b(?:\d[ -]*?){13,16}\b)|` + // Credit Cards
		`(Bearer\s+[a-zA-Z0-9\-\._~\+\/]+=*)`, // Bearer Tokens
	)
)

// Redacts PII patterns (email, SSN, credit card, bearer token) in val, leaving short or numeric values unchanged.
func RedactString(val string) string {
	if len(val) < 4 {
		return val
	}
	if isNumeric(val) {
		return val
	}
	return piiRegex.ReplaceAllString(val, "[REDACTED]")
}

// Redacts val when key matches a sensitive field name, or when val is a string containing PII patterns.
func RedactPair(key string, val any) any {
	if _, ok := sensitiveKeys[strings.ToLower(key)]; ok {
		return "[REDACTED]"
	}
	if s, ok := val.(string); ok && len(s) >= 4 && !isNumeric(s) {
		return piiRegex.ReplaceAllString(s, "[REDACTED]")
	}
	return val
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
