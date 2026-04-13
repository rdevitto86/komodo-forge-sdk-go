package httpcontext

import (
	"context"
	"testing"
)

func TestContextKeys_AreNonEmpty(t *testing.T) {
	keys := []ctxKey{
		START_TIME_KEY, END_TIME_KEY, DURATION_KEY,
		VERSION_KEY, URI_KEY, PATH_PARAMS_KEY, QUERY_PARAMS_KEY,
		VALIDATION_RULE_KEY, REQUEST_ID_KEY, REQUEST_TIMEOUT_KEY,
		CLIENT_IP_KEY, CLIENT_TYPE_KEY, USER_AGENT_KEY, METHOD_KEY,
		AUTH_VALID_KEY, SESSION_ID_KEY, SESSION_VALID_KEY, USER_ID_KEY,
		REQUEST_TYPE_KEY, SCOPES_KEY, IS_ADMIN_KEY, IDEMPOTENCY_KEY,
		IDEMPOTENCY_VALID_KEY, CORRELATION_ID_KEY, CSRF_TOKEN_KEY,
		CSRF_VALID_KEY, LOGGER_KEY, OTEL_LOGGER_KEY,
	}
	for _, k := range keys {
		if string(k) == "" {
			t.Errorf("context key is empty: %v", k)
		}
	}
}

func TestContextKeys_AreUnique(t *testing.T) {
	seen := make(map[ctxKey]bool)
	keys := []ctxKey{
		START_TIME_KEY, END_TIME_KEY, DURATION_KEY,
		VERSION_KEY, URI_KEY, PATH_PARAMS_KEY, QUERY_PARAMS_KEY,
		VALIDATION_RULE_KEY, REQUEST_ID_KEY, REQUEST_TIMEOUT_KEY,
		CLIENT_IP_KEY, CLIENT_TYPE_KEY, USER_AGENT_KEY, METHOD_KEY,
		AUTH_VALID_KEY, SESSION_ID_KEY, SESSION_VALID_KEY, USER_ID_KEY,
		REQUEST_TYPE_KEY, SCOPES_KEY, IS_ADMIN_KEY, IDEMPOTENCY_KEY,
		IDEMPOTENCY_VALID_KEY, CORRELATION_ID_KEY, CSRF_TOKEN_KEY,
		CSRF_VALID_KEY, LOGGER_KEY, OTEL_LOGGER_KEY,
	}
	for _, k := range keys {
		if seen[k] {
			t.Errorf("duplicate context key: %q", k)
		}
		seen[k] = true
	}
}

func TestContextKeys_CanBeUsedAsContextKeys_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, REQUEST_ID_KEY, "req-123")
	ctx = context.WithValue(ctx, USER_ID_KEY, "user-456")
	ctx = context.WithValue(ctx, AUTH_VALID_KEY, true)

	if val, ok := ctx.Value(REQUEST_ID_KEY).(string); !ok || val != "req-123" {
		t.Errorf("REQUEST_ID_KEY: got %v, want req-123", ctx.Value(REQUEST_ID_KEY))
	}
	if val, ok := ctx.Value(USER_ID_KEY).(string); !ok || val != "user-456" {
		t.Errorf("USER_ID_KEY: got %v, want user-456", ctx.Value(USER_ID_KEY))
	}
	if val, ok := ctx.Value(AUTH_VALID_KEY).(bool); !ok || !val {
		t.Errorf("AUTH_VALID_KEY: got %v, want true", ctx.Value(AUTH_VALID_KEY))
	}
}

func TestContextKeys_MissingKey_Failure(t *testing.T) {
	ctx := context.Background()
	if val := ctx.Value(SESSION_ID_KEY); val != nil {
		t.Errorf("expected nil for unset SESSION_ID_KEY, got %v", val)
	}
}

func TestContextKeys_Values(t *testing.T) {
	tests := []struct {
		key  ctxKey
		want string
	}{
		{START_TIME_KEY, "start_time"},
		{END_TIME_KEY, "end_time"},
		{DURATION_KEY, "duration"},
		{VERSION_KEY, "version"},
		{URI_KEY, "uri"},
		{PATH_PARAMS_KEY, "path_params"},
		{QUERY_PARAMS_KEY, "query_params"},
		{VALIDATION_RULE_KEY, "validation_rule"},
		{REQUEST_ID_KEY, "request_id"},
		{REQUEST_TIMEOUT_KEY, "request_timeout"},
		{CLIENT_IP_KEY, "client_ip"},
		{CLIENT_TYPE_KEY, "client_type"},
		{USER_AGENT_KEY, "user_agent"},
		{METHOD_KEY, "method"},
		{AUTH_VALID_KEY, "auth_valid"},
		{SESSION_ID_KEY, "session_id"},
		{SESSION_VALID_KEY, "session_valid"},
		{USER_ID_KEY, "user_id"},
		{REQUEST_TYPE_KEY, "request_type"},
		{SCOPES_KEY, "scopes"},
		{IS_ADMIN_KEY, "is_admin"},
		{IDEMPOTENCY_KEY, "idempotency_key"},
		{IDEMPOTENCY_VALID_KEY, "idempotency_key_valid"},
		{CORRELATION_ID_KEY, "correlation_id"},
		{CSRF_TOKEN_KEY, "csrf_token"},
		{CSRF_VALID_KEY, "csrf_token_valid"},
		{LOGGER_KEY, "logger"},
		{OTEL_LOGGER_KEY, "otel_logger"},
	}
	for _, tc := range tests {
		if string(tc.key) != tc.want {
			t.Errorf("key %q = %q, want %q", tc.key, string(tc.key), tc.want)
		}
	}
}
