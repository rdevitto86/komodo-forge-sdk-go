package headers

import (
	"net/http"
	"os"
	"strconv"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
)

func MaxContentLengthMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	limit := maxBytes
	if limit <= 0 {
		limit = DEFAULT_MAX_CONTENT_LENGTH
		if v := os.Getenv("MAX_CONTENT_LENGTH"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				limit = n
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
			if req.ContentLength > limit {
				httpErr.SendError(wtr, req, httpErr.Global.BadRequest,
					httpErr.WithStatus(http.StatusRequestEntityTooLarge),
					httpErr.WithMessage("Request entity too large"),
					httpErr.WithDetail("request body exceeds maximum allowed size"),
				)
				return
			}

			req.Body = http.MaxBytesReader(wtr, req.Body, limit)
			next.ServeHTTP(wtr, req)
		})
	}
}
