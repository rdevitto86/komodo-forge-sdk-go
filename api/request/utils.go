package request

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	httpcontext "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

// Returns the request ID from context (set by RequestIDMiddleware), falling back to the
// X-Request-ID header when the middleware has not run, then to "unknown".
func GetRequestID(req *http.Request) string {
	if rid, ok := req.Context().Value(httpcontext.REQUEST_ID_KEY).(string); ok && rid != "" {
		return rid
	}
	if rid := req.Header.Get("X-Request-ID"); rid != "" {
		return rid
	}
	return "unknown"
}

// Creates a unique request ID using random bytes encoded in hex.
func GenerateRequestId() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
