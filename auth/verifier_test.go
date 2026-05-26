package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// ── Unit Tests ───────────────────────────────────────────────────────────────

type testTokenClaims struct {
	Scopes  []string `json:"scp,omitempty"`
	IsAdmin bool     `json:"adm,omitempty"`
	gojwt.RegisteredClaims
}

func mustGenerateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return key
}

func signTestToken(t *testing.T, key *rsa.PrivateKey, kid string, claims gojwt.Claims) string {
	t.Helper()
	tok := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return s
}

func buildJWKSJSON(key *rsa.PublicKey, kid string) []byte {
	type jwkEntry struct {
		Kty string `json:"kty"`
		Use string `json:"use"`
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	type jwksPayload struct {
		Keys []jwkEntry `json:"keys"`
	}
	doc := jwksPayload{Keys: []jwkEntry{{
		Kty: "RSA",
		Use: "sig",
		Kid: kid,
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}}
	b, _ := json.Marshal(doc)
	return b
}

func newJWKSServer(t *testing.T, key *rsa.PublicKey, kid string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(buildJWKSJSON(key, kid))
	}))
}

func validClaims(subject, jti string, scopes []string) testTokenClaims {
	return testTokenClaims{
		Scopes: scopes,
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  gojwt.NewNumericDate(time.Now()),
			ID:        jti,
		},
	}
}

func assertError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("expected error %v, got %v", want, got)
	}
}

func TestNewJWKSVerifier_EmptyURL(t *testing.T) {
	_, err := NewJWKSVerifier(Config{})
	if err == nil {
		t.Fatal("expected error for empty JWKSURL, got nil")
	}
}

func TestJWKSVerifier_ValidToken(t *testing.T) {
	key := mustGenerateKey(t)
	server := newJWKSServer(t, &key.PublicKey, "kid-1")
	defer server.Close()

	v, err := NewJWKSVerifier(Config{JWKSURL: server.URL})
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	token := signTestToken(t, key, "kid-1", validClaims("user-abc", "jti-001", []string{"read:items"}))

	claims, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("expected successful verify, got: %v", err)
	}
	if claims == nil {
		t.Fatal("expected non-nil claims")
	}
	if claims.Subject != "user-abc" {
		t.Errorf("expected Subject=%q, got %q", "user-abc", claims.Subject)
	}
	if claims.JTI != "jti-001" {
		t.Errorf("expected JTI=%q, got %q", "jti-001", claims.JTI)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "read:items" {
		t.Errorf("expected Scopes=[read:items], got %v", claims.Scopes)
	}
}

func TestJWKSVerifier_ExpiredToken(t *testing.T) {
	key := mustGenerateKey(t)
	server := newJWKSServer(t, &key.PublicKey, "kid-exp")
	defer server.Close()

	v, err := NewJWKSVerifier(Config{JWKSURL: server.URL})
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	expired := testTokenClaims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "user-exp",
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  gojwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ID:        "jti-exp",
		},
	}
	token := signTestToken(t, key, "kid-exp", expired)

	_, err = v.Verify(context.Background(), token)
	assertError(t, err, ErrExpired)
}

func TestJWKSVerifier_TamperedSignature(t *testing.T) {
	keyA := mustGenerateKey(t)
	keyB := mustGenerateKey(t)

	// Server serves key B's public key but token is signed with key A.
	server := newJWKSServer(t, &keyB.PublicKey, "kid-tampered")
	defer server.Close()

	v, err := NewJWKSVerifier(Config{JWKSURL: server.URL})
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	token := signTestToken(t, keyA, "kid-tampered", validClaims("user-tampered", "jti-t", nil))

	_, err = v.Verify(context.Background(), token)
	assertError(t, err, ErrInvalidSignature)
}

func TestJWKSVerifier_StaleCache_RefetchSucceeds(t *testing.T) {
	keyA := mustGenerateKey(t)
	keyB := mustGenerateKey(t)

	var serveB int32 // 0 = serve key A, 1 = serve key B

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if atomic.LoadInt32(&serveB) == 0 {
			w.Write(buildJWKSJSON(&keyA.PublicKey, "kid-a"))
		} else {
			w.Write(buildJWKSJSON(&keyB.PublicKey, "kid-b"))
		}
	}))
	defer server.Close()

	v, err := NewJWKSVerifier(Config{JWKSURL: server.URL})
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	// Warm up the cache with key A.
	tokenA := signTestToken(t, keyA, "kid-a", validClaims("user-a", "jti-a", nil))
	if _, err := v.Verify(context.Background(), tokenA); err != nil {
		t.Fatalf("expected initial verify to succeed: %v", err)
	}

	// Rotate server to key B.
	atomic.StoreInt32(&serveB, 1)

	// Verify with key B — cache misses on kid-b, triggers re-fetch, should succeed.
	tokenB := signTestToken(t, keyB, "kid-b", validClaims("user-b", "jti-b", nil))
	claims, err := v.Verify(context.Background(), tokenB)
	if err != nil {
		t.Fatalf("expected re-fetch to succeed: %v", err)
	}
	if claims.Subject != "user-b" {
		t.Errorf("expected Subject=%q, got %q", "user-b", claims.Subject)
	}
}

func TestJWKSVerifier_UnknownKidAfterRefetch(t *testing.T) {
	key := mustGenerateKey(t)

	// Server only knows kid-a; token requests kid-b.
	server := newJWKSServer(t, &key.PublicKey, "kid-a")
	defer server.Close()

	v, err := NewJWKSVerifier(Config{JWKSURL: server.URL})
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	// Token carries kid-b which the server will never return.
	keyOther := mustGenerateKey(t)
	token := signTestToken(t, keyOther, "kid-b", validClaims("user-x", "jti-x", nil))

	_, err = v.Verify(context.Background(), token)
	assertError(t, err, ErrInvalidToken)
}

func TestJWKSVerifier_ServerUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // shut down immediately so the address is unreachable

	v, err := NewJWKSVerifier(Config{JWKSURL: url})
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	key := mustGenerateKey(t)
	token := signTestToken(t, key, "kid-gone", validClaims("user-gone", "jti-gone", nil))

	_, err = v.Verify(context.Background(), token)
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}
}
