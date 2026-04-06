package helpers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Creates a new HTTP request with optional body, headers, and context.
func NewRequest(
	method string,
	url string,
	body any,
	headers map[string]string,
	ctx context.Context,
) (*http.Request, error) {
	if url == "" { return nil, fmt.Errorf("url is required") }

	var bodyReader *strings.Reader
	switch method = strings.ToUpper(method); method {
		case "POST", "PUT", "PATCH":
			if str, ok := body.(string); ok {
				bodyReader = strings.NewReader(str)
			} else if jsonBytes, err := json.Marshal(body); err == nil {
				bodyReader = strings.NewReader(string(jsonBytes))
			} else {
				return nil, fmt.Errorf("error marshaling body: %v", err)
			}
		case "GET", "DELETE", "HEAD", "OPTIONS", "TRACE", "CONNECT":
			bodyReader = nil
		default:
			return nil, fmt.Errorf("invalid method: %s", method)
	}

	if ctx == nil { ctx = context.Background() }

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil { return nil, err }
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req, nil
}

// Creates a new HTTP request from an existing HTTP request.
func FromRequest(req *http.Request) (*http.Request, error) {
	if req == nil { return nil, fmt.Errorf("request is required") }
	return http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), req.Body)
}

// Creates a unique request ID using random bytes encoded in hex.
func GenerateRequestId() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
