package telemetry

import (
	"fmt"
	"net/http"
	"time"

	httpErr "github.com/rdevitto86/komodo-forge-sdk-go/api/errors"
	httpReq "github.com/rdevitto86/komodo-forge-sdk-go/api/request"
	httpRes "github.com/rdevitto86/komodo-forge-sdk-go/api/response"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

func TelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(wtr http.ResponseWriter, req *http.Request) {
		resWtr := &httpRes.ResponseWriter{ResponseWriter: wtr}
		start := time.Now()

		defer func() {
			ms := time.Since(start).Milliseconds()

			// Recover from panics and ensure a 500 is sent if nothing written.
			if rec := recover(); rec != nil {
				reqID := httpReq.GetRequestID(req)
				_ = reqID

				// Safely check status
				status := resWtr.Status
				if status == 0 {
					httpErr.SendError(
						wtr, req, httpErr.Global.Internal, httpErr.WithDetail("error occured while logging telemetry"),
					)
				}

				logger.Error("telemetry panicked!", fmt.Errorf("telemetry panicked: %v", rec))
				return
			}

			status := resWtr.Status
			bytesWritten := resWtr.BytesWritten
			if status == 0 {
				status = http.StatusOK
			}

			reqID := httpReq.GetRequestID(req)

			attrs := []any{
				logger.Attr("request_id", reqID),
				logger.Attr("method", req.Method),
				logger.Attr("path", req.URL.Path),
				logger.Attr("query", req.URL.RawQuery),
				logger.Attr("status", status),
				logger.Attr("bytes", bytesWritten),
				logger.Attr("latency_ms", ms),
				logger.Attr("ip", req.RemoteAddr),
				logger.Attr("user_agent", req.UserAgent()),
				logger.Attr("referer", req.Referer()),
				logger.Attr("proto", req.Proto),
				logger.Attr("host", req.Host),
				logger.Attr("start_time", start.UTC().Format(time.RFC3339Nano)),
				logger.Attr("finish_time", time.Now().UTC().Format(time.RFC3339Nano)),
			}

			if status >= 400 {
				logger.Error("request failed", fmt.Errorf("status %d", status), attrs...)
			} else {
				logger.Info("request completed", attrs...)
			}
		}()

		next.ServeHTTP(resWtr, req)
	})
}
