package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/rdevitto86/komodo-forge-sdk-go/config"
	"github.com/rdevitto86/komodo-forge-sdk-go/crypto/jwt"
	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// testRSAKey holds the generated RSA key pair for the test run.
var testRSAKey *rsa.PrivateKey

// TestMain generates a single RSA-2048 key pair, initializes the jwt package
// once, and then runs all tests. Using a package-level key avoids the
// keysInitialized once-flag from blocking re-initialization.
func TestMain(m *testing.M) {
	var err error
	testRSAKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate RSA key: " + err.Error())
	}

	privDER := x509.MarshalPKCS1PrivateKey(testRSAKey)
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}))

	pubDER, err := x509.MarshalPKIXPublicKey(&testRSAKey.PublicKey)
	if err != nil {
		panic("failed to marshal public key: " + err.Error())
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	config.SetConfigValue("JWT_PRIVATE_KEY", privPEM)
	config.SetConfigValue("JWT_PUBLIC_KEY", pubPEM)
	config.SetConfigValue("JWT_ISSUER", "test-issuer")
	config.SetConfigValue("JWT_AUDIENCE", "test-audience")
	config.SetConfigValue("JWT_KID", "test-kid")
	os.Setenv("JWT_PRIVATE_KEY", privPEM)
	os.Setenv("JWT_PUBLIC_KEY", pubPEM)
	os.Setenv("JWT_ISSUER", "test-issuer")
	os.Setenv("JWT_AUDIENCE", "test-audience")
	os.Setenv("JWT_KID", "test-kid")

	if err := jwt.InitializeKeys(); err != nil {
		panic("failed to initialize JWT keys: " + err.Error())
	}

	os.Exit(m.Run())
}

// signCustomToken creates a signed RS256 JWT with the provided claims using
// the test RSA key. This allows testing specific claim combinations (e.g. IsAdmin).
func signCustomToken(t *testing.T, claims gojwt.Claims) string {
	t.Helper()
	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	signed, err := token.SignedString(testRSAKey)
	if err != nil {
		t.Fatalf("failed to sign custom token: %v", err)
	}
	return signed
}

// --- Authorization Middleware Tests ---

func TestAuthMiddleware_Success(t *testing.T) {
	tokenStr, err := jwt.SignToken("test-issuer", "user-123", "test-audience", 3600, nil)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		authValid, _ := r.Context().Value(ctxKeys.AUTH_VALID_KEY).(bool)
		if !authValid {
			t.Error("expected AUTH_VALID_KEY to be true in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	AuthMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called with valid token")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// TestAuthMiddleware_SetsSubjectAndSessionID verifies that Subject and ID claims
// populate USER_ID_KEY and SESSION_ID_KEY in context.
func TestAuthMiddleware_SetsSubjectAndSessionID(t *testing.T) {
	tokenStr, err := jwt.SignToken("test-issuer", "subject-abc", "test-audience", 3600, nil)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	var userID, sessionID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ = r.Context().Value(ctxKeys.USER_ID_KEY).(string)
		sessionID, _ = r.Context().Value(ctxKeys.SESSION_ID_KEY).(string)
		w.WriteHeader(http.StatusOK)
	})

	AuthMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if userID != "subject-abc" {
		t.Errorf("expected USER_ID_KEY='subject-abc', got %q", userID)
	}
	if sessionID == "" {
		t.Error("expected SESSION_ID_KEY to be set (JWT ID)")
	}
}

// TestAuthMiddleware_WithAPIScopes verifies that scopes populate REQUEST_TYPE_KEY=api
// and SCOPES_KEY in context.
func TestAuthMiddleware_WithAPIScopes(t *testing.T) {
	tokenStr, err := jwt.SignToken("test-issuer", "user-api", "test-audience", 3600, []string{"read:items", "write:items"})
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	var reqType string
	var scopes []string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqType, _ = r.Context().Value(ctxKeys.REQUEST_TYPE_KEY).(string)
		scopes, _ = r.Context().Value(ctxKeys.SCOPES_KEY).([]string)
		w.WriteHeader(http.StatusOK)
	})

	AuthMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if reqType != "api" {
		t.Errorf("expected REQUEST_TYPE_KEY='api', got %q", reqType)
	}
	if len(scopes) == 0 {
		t.Error("expected SCOPES_KEY to be populated")
	}
}

// TestAuthMiddleware_WithIsAdminClaim verifies that IsAdmin=true sets IS_ADMIN_KEY
// and REQUEST_TYPE_KEY="ui" (admin with no scopes is treated as UI).
func TestAuthMiddleware_WithIsAdminClaim(t *testing.T) {
	type adminClaims struct {
		Scopes  []string `json:"scp,omitempty"`
		IsAdmin bool     `json:"adm,omitempty"`
		gojwt.RegisteredClaims
	}

	claims := adminClaims{
		IsAdmin: true,
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "admin-user",
			Issuer:    "test-issuer",
			Audience:  gojwt.ClaimStrings{"test-audience"},
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  gojwt.NewNumericDate(time.Now()),
			NotBefore: gojwt.NewNumericDate(time.Now()),
			ID:        "admin-session-id",
		},
	}
	tokenStr := signCustomToken(t, claims)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	var isAdmin bool
	var reqType string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, _ = r.Context().Value(ctxKeys.IS_ADMIN_KEY).(bool)
		reqType, _ = r.Context().Value(ctxKeys.REQUEST_TYPE_KEY).(string)
		w.WriteHeader(http.StatusOK)
	})

	AuthMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if !isAdmin {
		t.Error("expected IS_ADMIN_KEY to be true")
	}
	if reqType != "ui" {
		t.Errorf("expected REQUEST_TYPE_KEY='ui' for admin with no scopes, got %q", reqType)
	}
}

// Auth middleware requires JWT keys to be initialized; without them any
// token extraction succeeds but validation fails with a 401.

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_NonBearerAuthorization(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	AuthMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_BearerTokenKeysNotInitialized(t *testing.T) {
	// JWT keys are not initialized in test env; token extraction succeeds
	// but ValidateToken returns (false, err) → 401.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer some.jwt.token")
	rec := httptest.NewRecorder()

	AuthMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- Scope Middleware Tests ---

func requestWithScopes(scopes []string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if scopes != nil {
		ctx := context.WithValue(req.Context(), ctxKeys.SCOPES_KEY, scopes)
		req = req.WithContext(ctx)
	}
	return req
}

func TestRequireServiceScope_NoScopes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	RequireServiceScope(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 when no scopes in context, got %d", rec.Code)
	}
}

func TestRequireServiceScope_EmptyScopes(t *testing.T) {
	req := requestWithScopes([]string{})
	rec := httptest.NewRecorder()

	RequireServiceScope(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for empty scopes, got %d", rec.Code)
	}
}

func TestRequireServiceScope_NoServiceScopePrefix(t *testing.T) {
	req := requestWithScopes([]string{"read:items", "write:items"})
	rec := httptest.NewRecorder()

	RequireServiceScope(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 when no svc: prefix, got %d", rec.Code)
	}
}

func TestRequireServiceScope_WithServiceScope(t *testing.T) {
	req := requestWithScopes([]string{"read:items", "svc:inventory"})
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	RequireServiceScope(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called when svc: scope present")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRequireServiceScope_OnlyServiceScope(t *testing.T) {
	req := requestWithScopes([]string{"svc:auth"})
	rec := httptest.NewRecorder()

	RequireServiceScope(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for svc: only scope, got %d", rec.Code)
	}
}
