package httpcontext

import "context"

// Returns a copy of ctx carrying the request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, REQUEST_ID_KEY, id)
}

// Returns a copy of ctx carrying the correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CORRELATION_ID_KEY, id)
}

// Returns a copy of ctx carrying the authenticated subject (user ID).
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, USER_ID_KEY, id)
}

// Returns a copy of ctx carrying the session ID.
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SESSION_ID_KEY, id)
}

// Returns a copy of ctx carrying the verified client type ("browser" or "api").
func WithClientType(ctx context.Context, clientType string) context.Context {
	return context.WithValue(ctx, CLIENT_TYPE_KEY, clientType)
}

// Returns a copy of ctx carrying the request type ("ui" or "api").
func WithRequestType(ctx context.Context, requestType string) context.Context {
	return context.WithValue(ctx, REQUEST_TYPE_KEY, requestType)
}

// Returns a copy of ctx carrying the client IP.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, CLIENT_IP_KEY, ip)
}

// Returns a copy of ctx carrying the CSRF token.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, CSRF_TOKEN_KEY, token)
}

// Returns a copy of ctx carrying the scopes.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, SCOPES_KEY, scopes)
}

// Returns a copy of ctx marking the request as carrying a valid authenticated token.
func WithAuthValid(ctx context.Context, valid bool) context.Context {
	return context.WithValue(ctx, AUTH_VALID_KEY, valid)
}

// Returns a copy of ctx marking the authenticated principal as admin.
func WithAdmin(ctx context.Context, admin bool) context.Context {
	return context.WithValue(ctx, IS_ADMIN_KEY, admin)
}

// Returns a copy of ctx marking the request as having passed CSRF validation.
func WithCSRFValid(ctx context.Context, valid bool) context.Context {
	return context.WithValue(ctx, CSRF_VALID_KEY, valid)
}

// Returns a copy of ctx marking the request as carrying a valid idempotency key.
func WithIdempotencyValid(ctx context.Context, valid bool) context.Context {
	return context.WithValue(ctx, IDEMPOTENCY_VALID_KEY, valid)
}
