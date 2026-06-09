package httpcontext

import "context"

// stringValue returns the string stored at key, or "" if absent or of another type.
func stringValue(ctx context.Context, key ctxKey) string {
	v, _ := ctx.Value(key).(string)
	return v
}

// boolValue returns the bool stored at key, or false if absent or of another type.
func boolValue(ctx context.Context, key ctxKey) bool {
	v, _ := ctx.Value(key).(bool)
	return v
}

// Returns the request ID stored in ctx, or "" if absent.
func GetRequestID(ctx context.Context) string { return stringValue(ctx, REQUEST_ID_KEY) }

// Returns the correlation ID stored in ctx, or "" if absent.
func GetCorrelationID(ctx context.Context) string { return stringValue(ctx, CORRELATION_ID_KEY) }

// Returns the authenticated subject (user ID) stored in ctx, or "" if absent.
func GetUserID(ctx context.Context) string { return stringValue(ctx, USER_ID_KEY) }

// Returns the session ID (JWT ID) stored in ctx, or "" if absent.
func GetSessionID(ctx context.Context) string { return stringValue(ctx, SESSION_ID_KEY) }

// Returns the verified client type ("browser" or "api") stored in ctx, or "" if absent.
func GetClientType(ctx context.Context) string { return stringValue(ctx, CLIENT_TYPE_KEY) }

// Returns the request type ("ui" or "api") stored in ctx, or "" if absent.
func GetRequestType(ctx context.Context) string { return stringValue(ctx, REQUEST_TYPE_KEY) }

// Returns the client IP stored in ctx, or "" if absent.
func GetClientIP(ctx context.Context) string { return stringValue(ctx, CLIENT_IP_KEY) }

// Returns the User-Agent stored in ctx, or "" if absent.
func GetUserAgent(ctx context.Context) string { return stringValue(ctx, USER_AGENT_KEY) }

// Returns the HTTP method stored in ctx, or "" if absent.
func GetMethod(ctx context.Context) string { return stringValue(ctx, METHOD_KEY) }

// Returns the request URI stored in ctx, or "" if absent.
func GetURI(ctx context.Context) string { return stringValue(ctx, URI_KEY) }

// Returns the API version stored in ctx, or "" if absent.
func GetVersion(ctx context.Context) string { return stringValue(ctx, VERSION_KEY) }

// Returns the CSRF token stored in ctx, or "" if absent.
func GetCSRFToken(ctx context.Context) string { return stringValue(ctx, CSRF_TOKEN_KEY) }

// Returns the idempotency key stored in ctx, or "" if absent.
func GetIdempotencyKey(ctx context.Context) string { return stringValue(ctx, IDEMPOTENCY_KEY) }

// Returns the scopes stored in ctx, or nil if absent.
func GetScopes(ctx context.Context) []string {
	v, _ := ctx.Value(SCOPES_KEY).([]string)
	return v
}

// Returns the path parameters stored in ctx, or nil if absent.
func GetPathParams(ctx context.Context) map[string]string {
	v, _ := ctx.Value(PATH_PARAMS_KEY).(map[string]string)
	return v
}

// Returns the query parameters stored in ctx, or nil if absent.
func GetQueryParams(ctx context.Context) map[string]string {
	v, _ := ctx.Value(QUERY_PARAMS_KEY).(map[string]string)
	return v
}

// Reports whether the request carried a valid authenticated token.
func IsAuthValid(ctx context.Context) bool { return boolValue(ctx, AUTH_VALID_KEY) }

// Reports whether the authenticated principal is an admin.
func IsAdmin(ctx context.Context) bool { return boolValue(ctx, IS_ADMIN_KEY) }

// Reports whether the request carried a valid session.
func IsSessionValid(ctx context.Context) bool { return boolValue(ctx, SESSION_VALID_KEY) }

// Reports whether the request passed CSRF validation.
func IsCSRFValid(ctx context.Context) bool { return boolValue(ctx, CSRF_VALID_KEY) }

// Reports whether the request carried a valid idempotency key.
func IsIdempotencyValid(ctx context.Context) bool { return boolValue(ctx, IDEMPOTENCY_VALID_KEY) }
