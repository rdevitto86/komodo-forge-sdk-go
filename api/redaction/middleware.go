package redaction

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	redact "github.com/rdevitto86/komodo-forge-sdk-go/security/redaction"
)

func RedactionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		r2 := new(http.Request)
		*r2 = *req

		r2.Header = redactHeaders(req.Header)

		if req.URL != nil {
			u := *req.URL
			u.RawQuery = redactQuery(req.URL.Query()).Encode()
			r2.URL = &u
		}

		if req.Body != nil {
			b, err := io.ReadAll(req.Body)
			if err == nil {
				req.Body = io.NopCloser(bytes.NewReader(b))
				ct := req.Header.Get("Content-Type")
				rb := redactBody(b, ct)
				r2.Body = io.NopCloser(bytes.NewReader(rb))
			}
		}
		next.ServeHTTP(wtr, r2)
	})
}

func redactHeaders(header http.Header) http.Header {
	if header == nil {
		return nil
	}

	out := make(http.Header, len(header))
	for k, val := range header {
		if redact.IsSensitiveKey(k) {
			out[k] = []string{"REDACTED"}
			continue
		}
		newVals := make([]string, 0, len(val))
		for _, v := range val {
			if looksLikeToken(v) {
				newVals = append(newVals, "REDACTED")
			} else {
				newVals = append(newVals, v)
			}
		}
		out[k] = newVals
	}
	return out
}

var (
	bearerRE    = regexp.MustCompile(`(?i)^\s*bearer\s+[A-Za-z0-9\-\._~\+/]+=*$`)
	longTokenRE = regexp.MustCompile(`[A-Za-z0-9\-\._~\+/]{20,}`)
)

func looksLikeToken(s string) bool {
	if s == "" {
		return false
	}
	if bearerRE.MatchString(s) {
		return true
	}
	if longTokenRE.MatchString(s) && len(s) > 30 {
		return true
	}
	return false
}

func redactQuery(vals url.Values) url.Values {
	if vals == nil {
		return nil
	}

	out := url.Values{}
	for k, v := range vals {
		if redact.IsSensitiveKey(k) {
			out[k] = []string{"REDACTED"}
			continue
		}

		nameVal := make([]string, 0, len(v))
		for _, vv := range v {
			if looksLikeToken(vv) {
				nameVal = append(nameVal, "REDACTED")
			} else {
				nameVal = append(nameVal, vv)
			}
		}
		out[k] = nameVal
	}
	return out
}

func redactBody(b []byte, contentType string) []byte {
	if len(b) == 0 {
		return b
	}

	if strings.Contains(strings.ToLower(contentType), "application/json") {
		var v any
		if err := json.Unmarshal(b, &v); err == nil {
			v = redact.RedactValue(v)
			if out, err := json.Marshal(v); err == nil {
				return out
			}
		}
	}

	rb := bearerRE.ReplaceAllStringFunc(string(b), func(_ string) string { return "REDACTED" })
	rb = longTokenRE.ReplaceAllString(rb, "REDACTED")

	return []byte(rb)
}
