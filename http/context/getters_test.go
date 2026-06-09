package httpcontext

import (
	"context"
	"reflect"
	"testing"
)

func TestStringAccessors_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		set  func(context.Context, string) context.Context
		get  func(context.Context) string
	}{
		{"request ID", WithRequestID, GetRequestID},
		{"correlation ID", WithCorrelationID, GetCorrelationID},
		{"user ID", WithUserID, GetUserID},
		{"session ID", WithSessionID, GetSessionID},
		{"client type", WithClientType, GetClientType},
		{"request type", WithRequestType, GetRequestType},
		{"client IP", WithClientIP, GetClientIP},
		{"csrf token", WithCSRFToken, GetCSRFToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.get(context.Background()); got != "" {
				t.Errorf("expected empty default, got %q", got)
			}
			ctx := tc.set(context.Background(), "value-"+tc.name)
			if got := tc.get(ctx); got != "value-"+tc.name {
				t.Errorf("got %q, want %q", got, "value-"+tc.name)
			}
		})
	}
}

func TestBoolAccessors_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		set  func(context.Context, bool) context.Context
		get  func(context.Context) bool
	}{
		{"auth valid", WithAuthValid, IsAuthValid},
		{"admin", WithAdmin, IsAdmin},
		{"csrf valid", WithCSRFValid, IsCSRFValid},
		{"idempotency valid", WithIdempotencyValid, IsIdempotencyValid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.get(context.Background()) {
				t.Error("expected false default")
			}
			if !tc.get(tc.set(context.Background(), true)) {
				t.Error("expected true after set")
			}
		})
	}
}

func TestScopesAccessor_RoundTrip(t *testing.T) {
	if got := GetScopes(context.Background()); got != nil {
		t.Errorf("expected nil default, got %v", got)
	}
	want := []string{"read:items", "svc:internal"}
	ctx := WithScopes(context.Background(), want)
	if got := GetScopes(ctx); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestGetters_WrongTypeReturnsZero(t *testing.T) {
	// A value stored under a key with an unexpected type must not panic; it returns the zero value.
	ctx := context.WithValue(context.Background(), USER_ID_KEY, 12345)
	if got := GetUserID(ctx); got != "" {
		t.Errorf("expected empty string for wrong-typed value, got %q", got)
	}
}
