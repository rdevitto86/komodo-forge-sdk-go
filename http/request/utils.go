package request

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	httpcontext "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func GetRequestID(req *http.Request) string {
	if rid, ok := req.Context().Value(httpcontext.REQUEST_ID_KEY).(string); ok && rid != "" {
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
