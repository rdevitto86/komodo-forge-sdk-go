package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	golangJWT "github.com/golang-jwt/jwt/v5"
)

func generateTestKeys(t *testing.T) (priv *rsa.PrivateKey, privPEM, pubPEM string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}))
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}))
	return
}

func newTestClient(t *testing.T) (*Client, *rsa.PrivateKey) {
	t.Helper()
	priv, privPEM, pubPEM := generateTestKeys(t)
	c, err := New(context.Background(), Config{
		PrivateKeyPEM: privPEM,
		PublicKeyPEM:  pubPEM,
		KID:           "test-kid",
		Issuer:        "test-issuer",
		Audience:      "test-audience",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, priv
}

func signCustom(t *testing.T, priv *rsa.PrivateKey, claims CustomClaims) string {
	t.Helper()
	tok := golangJWT.NewWithClaims(golangJWT.SigningMethodRS256, claims)
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("signCustom: %v", err)
	}
	return s
}

type fakeSecrets struct {
	pem string
	err error
}

func (f fakeSecrets) GetSecret(ctx context.Context, name string) (string, error) {
	return f.pem, f.err
}

func TestNew_Success(t *testing.T) {
	c, _ := newTestClient(t)
	if c.privateKey == nil || c.publicKey == nil {
		t.Error("expected both keys to be set")
	}
	if c.leeway != defaultLeeway {
		t.Errorf("leeway = %v, want default %v", c.leeway, defaultLeeway)
	}
}

func TestNew_MissingPublicKey_Failure(t *testing.T) {
	_, err := New(context.Background(), Config{Issuer: "i", Audience: "a"})
	if err == nil {
		t.Error("expected error for missing public key")
	}
}

func TestNew_MissingIssuer_Failure(t *testing.T) {
	_, _, pub := generateTestKeys(t)
	_, err := New(context.Background(), Config{PublicKeyPEM: pub, Audience: "a"})
	if err == nil {
		t.Error("expected error for missing issuer")
	}
}

func TestNew_MissingAudience_Failure(t *testing.T) {
	_, _, pub := generateTestKeys(t)
	_, err := New(context.Background(), Config{PublicKeyPEM: pub, Issuer: "i"})
	if err == nil {
		t.Error("expected error for missing audience")
	}
}

func TestNew_InvalidPublicKeyPEM_Failure(t *testing.T) {
	_, err := New(context.Background(), Config{PublicKeyPEM: "not-pem", Issuer: "i", Audience: "a"})
	if err == nil {
		t.Error("expected error for invalid public key PEM")
	}
}

func TestNew_InvalidPrivateKeyPEM_Failure(t *testing.T) {
	_, _, pub := generateTestKeys(t)
	_, err := New(context.Background(), Config{PrivateKeyPEM: "not-pem", PublicKeyPEM: pub, Issuer: "i", Audience: "a"})
	if err == nil {
		t.Error("expected error for invalid private key PEM")
	}
}

func TestNew_MismatchedKeyPair_Failure(t *testing.T) {
	_, privPEM, _ := generateTestKeys(t)
	_, _, otherPub := generateTestKeys(t)
	_, err := New(context.Background(), Config{PrivateKeyPEM: privPEM, PublicKeyPEM: otherPub, Issuer: "i", Audience: "a"})
	if err == nil {
		t.Error("expected error for mismatched key pair")
	}
}

func TestNew_VerifyOnly_Success(t *testing.T) {
	_, _, pub := generateTestKeys(t)
	c, err := New(context.Background(), Config{PublicKeyPEM: pub, Issuer: "i", Audience: "a"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.privateKey != nil {
		t.Error("expected no private key on verify-only client")
	}
}

func TestNew_SecretsProvider_Success(t *testing.T) {
	_, privPEM, pubPEM := generateTestKeys(t)
	c, err := New(context.Background(), Config{
		PublicKeyPEM:   pubPEM,
		Issuer:         "test-issuer",
		Audience:       "test-audience",
		Secrets:        fakeSecrets{pem: privPEM},
		PrivateKeyName: "jwt-private-key",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.SignToken("test-issuer", "u", "test-audience", 3600, nil); err != nil {
		t.Errorf("SignToken with secrets-loaded key: %v", err)
	}
}

func TestNew_SecretsProvider_Error(t *testing.T) {
	_, _, pubPEM := generateTestKeys(t)
	_, err := New(context.Background(), Config{
		PublicKeyPEM:   pubPEM,
		Issuer:         "i",
		Audience:       "a",
		Secrets:        fakeSecrets{err: errors.New("boom")},
		PrivateKeyName: "jwt-private-key",
	})
	if err == nil {
		t.Error("expected error when secrets provider fails")
	}
}

func TestSignToken_Success(t *testing.T) {
	c, _ := newTestClient(t)
	token, err := c.SignToken("test-issuer", "user-123", "test-audience", 3600, []string{"read", "write"})
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestSignToken_NoPrivateKey_Failure(t *testing.T) {
	_, _, pub := generateTestKeys(t)
	c, _ := New(context.Background(), Config{PublicKeyPEM: pub, Issuer: "i", Audience: "a"})
	if _, err := c.SignToken("i", "u", "a", 3600, nil); err == nil {
		t.Error("expected error signing without a private key")
	}
}

func TestSignToken_NonPositiveTTL_Failure(t *testing.T) {
	c, _ := newTestClient(t)
	for _, ttl := range []int64{0, -1} {
		if _, err := c.SignToken("test-issuer", "u", "test-audience", ttl, nil); err == nil {
			t.Errorf("expected error for ttl=%d", ttl)
		}
	}
}

func TestValidateToken_Success(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("test-issuer", "u", "test-audience", 3600, []string{"read"})
	ok, err := c.ValidateToken(token)
	if err != nil || !ok {
		t.Fatalf("ValidateToken = %v, %v; want true, nil", ok, err)
	}
}

func TestValidateToken_Expired_Failure(t *testing.T) {
	c, priv := newTestClient(t)
	now := time.Now()
	token := signCustom(t, priv, CustomClaims{RegisteredClaims: golangJWT.RegisteredClaims{
		Issuer:    "test-issuer",
		Audience:  golangJWT.ClaimStrings{"test-audience"},
		ExpiresAt: golangJWT.NewNumericDate(now.Add(-5 * time.Minute)),
		IssuedAt:  golangJWT.NewNumericDate(now.Add(-10 * time.Minute)),
	}})
	if _, err := c.ValidateToken(token); err == nil {
		t.Error("expected error for expired token")
	}
}

func TestValidateToken_WrongIssuer_Failure(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("wrong-issuer", "u", "test-audience", 3600, nil)
	if _, err := c.ValidateToken(token); err == nil {
		t.Error("expected error for wrong issuer")
	}
}

func TestValidateToken_WrongAudience_Failure(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("test-issuer", "u", "wrong-audience", 3600, nil)
	if _, err := c.ValidateToken(token); err == nil {
		t.Error("expected error for wrong audience")
	}
}

func TestValidateToken_Tampered_Failure(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("test-issuer", "u", "test-audience", 3600, nil)
	parts := strings.Split(token, ".")
	parts[2] = "AAAA" + parts[2]
	if _, err := c.ValidateToken(strings.Join(parts, ".")); err == nil {
		t.Error("expected error for tampered signature")
	}
}

func TestValidateToken_WrongMethod_Failure(t *testing.T) {
	c, _ := newTestClient(t)
	tok := golangJWT.NewWithClaims(golangJWT.SigningMethodHS256, golangJWT.MapClaims{
		"iss": "test-issuer",
		"aud": golangJWT.ClaimStrings{"test-audience"},
	})
	hs, _ := tok.SignedString([]byte("secret"))
	if _, err := c.ValidateToken(hs); err == nil {
		t.Error("expected error for HMAC-signed token")
	}
}

func TestValidateAndParseClaims_Success(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("test-issuer", "user-456", "test-audience", 3600, []string{"read", "write"})
	claims, err := c.ValidateAndParseClaims(token)
	if err != nil {
		t.Fatalf("ValidateAndParseClaims: %v", err)
	}
	if claims.Subject != "user-456" {
		t.Errorf("Subject = %q, want user-456", claims.Subject)
	}
	if len(claims.Scopes) != 2 {
		t.Errorf("Scopes = %v, want 2", claims.Scopes)
	}
}

func TestValidateAndParseClaims_Expired_Failure(t *testing.T) {
	c, priv := newTestClient(t)
	now := time.Now()
	token := signCustom(t, priv, CustomClaims{RegisteredClaims: golangJWT.RegisteredClaims{
		Issuer:    "test-issuer",
		Audience:  golangJWT.ClaimStrings{"test-audience"},
		ExpiresAt: golangJWT.NewNumericDate(now.Add(-5 * time.Minute)),
	}})
	if _, err := c.ValidateAndParseClaims(token); err == nil {
		t.Error("expected error for expired token")
	}
}

func TestValidateAndParseClaims_WrongIssuer_Failure(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("wrong-issuer", "u", "test-audience", 3600, nil)
	if _, err := c.ValidateAndParseClaims(token); err == nil {
		t.Error("expected error for wrong issuer")
	}
}

func TestParseClaims_Success_IgnoresIssuerAudience(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("some-other-issuer", "user-789", "some-other-audience", 3600, nil)
	claims, err := c.ParseClaims(token)
	if err != nil {
		t.Fatalf("ParseClaims: %v", err)
	}
	if claims.Subject != "user-789" {
		t.Errorf("Subject = %q, want user-789", claims.Subject)
	}
}

func TestParseClaims_WrongMethod_Failure(t *testing.T) {
	c, _ := newTestClient(t)
	tok := golangJWT.NewWithClaims(golangJWT.SigningMethodHS256, golangJWT.MapClaims{"sub": "u"})
	hs, _ := tok.SignedString([]byte("secret"))
	if _, err := c.ParseClaims(hs); err == nil {
		t.Error("expected error for HMAC-signed token")
	}
}

func TestLeeway_WithinTolerancePasses(t *testing.T) {
	_, privPEM, pubPEM := generateTestKeys(t)
	c, err := New(context.Background(), Config{PrivateKeyPEM: privPEM, PublicKeyPEM: pubPEM, Issuer: "test-issuer", Audience: "test-audience", Leeway: 30 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	priv, _ := golangJWT.ParseRSAPrivateKeyFromPEM([]byte(privPEM))
	now := time.Now()
	token := signCustom(t, priv, CustomClaims{RegisteredClaims: golangJWT.RegisteredClaims{
		Issuer:    "test-issuer",
		Audience:  golangJWT.ClaimStrings{"test-audience"},
		ExpiresAt: golangJWT.NewNumericDate(now.Add(-10 * time.Second)),
		IssuedAt:  golangJWT.NewNumericDate(now.Add(-1 * time.Minute)),
	}})
	if ok, err := c.ValidateToken(token); err != nil || !ok {
		t.Errorf("expected token expired within leeway to pass, got %v, %v", ok, err)
	}
}

func TestLeeway_BeyondToleranceFails(t *testing.T) {
	c, priv := newTestClient(t)
	now := time.Now()
	token := signCustom(t, priv, CustomClaims{RegisteredClaims: golangJWT.RegisteredClaims{
		Issuer:    "test-issuer",
		Audience:  golangJWT.ClaimStrings{"test-audience"},
		ExpiresAt: golangJWT.NewNumericDate(now.Add(-2 * time.Minute)),
		IssuedAt:  golangJWT.NewNumericDate(now.Add(-5 * time.Minute)),
	}})
	if _, err := c.ValidateToken(token); err == nil {
		t.Error("expected token expired beyond leeway to fail")
	}
}

func TestExtractTokenFromRequest_Success(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-test-token")
	token, err := ExtractTokenFromRequest(req)
	if err != nil || token != "my-test-token" {
		t.Errorf("got %q, %v; want my-test-token, nil", token, err)
	}
}

func TestExtractTokenFromRequest_CaseInsensitiveScheme(t *testing.T) {
	for _, scheme := range []string{"bearer", "BEARER", "BeArEr"} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", scheme+" tok")
		token, err := ExtractTokenFromRequest(req)
		if err != nil || token != "tok" {
			t.Errorf("scheme %q: got %q, %v; want tok, nil", scheme, token, err)
		}
	}
}

func TestExtractTokenFromRequest_TrimsWhitespace(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "  Bearer   spaced-token  ")
	token, err := ExtractTokenFromRequest(req)
	if err != nil || token != "spaced-token" {
		t.Errorf("got %q, %v; want spaced-token, nil", token, err)
	}
}

func TestExtractTokenFromRequest_NoHeader_Failure(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if _, err := ExtractTokenFromRequest(req); err == nil {
		t.Error("expected error for missing header")
	}
}

func TestExtractTokenFromRequest_NonBearer_Failure(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	if _, err := ExtractTokenFromRequest(req); err == nil {
		t.Error("expected error for non-Bearer header")
	}
}

func TestConcurrentValidate(t *testing.T) {
	c, _ := newTestClient(t)
	token, _ := c.SignToken("test-issuer", "u", "test-audience", 3600, []string{"read"})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.ValidateAndParseClaims(token); err != nil {
				t.Errorf("concurrent validate: %v", err)
			}
		}()
	}
	wg.Wait()
}
