package ratelimit

import (
	"fmt"
	"net/http"
	"strconv"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/api/request"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

func RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		key := httpReq.GetClientKey(req)
		allowed, wait, err := Allow(req.Context(), key)

		if err != nil {
			if ShouldFailOpen() {
				logger.Error("rate limiter failing open for client: "+key, err)
			} else {
				logger.Error("rate limiter failed for client: "+key, err)
				httpErr.SendError(
					wtr, req, httpErr.Global.Internal, httpErr.WithDetail("internal rate limiter error"),
				)
				return
			}
		} else if !allowed {
			if wait > 0 {
				wtr.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds()+0.5)))
			}
			logger.Error("rate limit exceeded for client: "+key, fmt.Errorf("rate limit exceeded"))
			httpErr.SendError(
				wtr, req, httpErr.Global.TooManyRequests, httpErr.WithDetail("rate limit exceeded"),
			)
			return
		}

		next.ServeHTTP(wtr, req)
	})
}
