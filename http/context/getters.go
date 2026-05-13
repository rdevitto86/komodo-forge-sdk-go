package httpcontext

import "context"

// GetCorrelationID returns the correlation ID stored in ctx, or "" if absent.
func GetCorrelationID(ctx context.Context) string {
	v, _ := ctx.Value(CORRELATION_ID_KEY).(string)
	return v
}