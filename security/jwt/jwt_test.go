package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"testing"

	golangJWT "github.com/golang-jwt/jwt/v5"
)

// generateTestKeys creates a fresh RSA-2048 key pair and returns PEM-encoded strings.
func generateTestKeys(t *testing.T) (privPEM, pubPEM string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generateTestKeys: %v", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}))

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("generateTestKeys: marshal public key: %v", err)
	}
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}))
	return
}

// resetJWTState wipes all package-level JWT state so each test starts clean.
func resetJWTState() {
	keyMutex.Lock()
	defer keyMutex.Unlock()
	keysInitialized = false
	cachedPrivateKey = nil
	cachedPublicKey = nil
	kid = ""
	iss = ""
	aud = ""
}

// setupInitializedKeys sets up config + calls InitializeKeys, cleaning up on t.Cleanup.
func setupInitializedKeys(t *testing.T) {
	t.Helper()
	resetJWTState()
	privPEM, pubPEM := generateTestKeys(t)
	os.Setenv("JWT_PRIVATE_KEY", privPEM)
	os.Setenv("JWT_PUBLIC_KEY", pubPEM)
	os.Setenv("JWT_KID", "test-kid")
	os.Setenv("JWT_ISSUER", "test-issuer")
	os.Setenv("JWT_AUDIENCE", "test-audience")
	t.Cleanup(func() {
		os.Unsetenv("JWT_PRIVATE_KEY")
		os.Unsetenv("JWT_PUBLIC_KEY")
		os.Unsetenv("JWT_KID")
		os.Unsetenv("JWT_ISSUER")
		os.Unsetenv("JWT_AUDIENCE")
		resetJWTState()
	})
	if err := InitializeKeys(); err != nil {
		t.Fatalf("setupInitializedKeys: InitializeKeys: %v", err)
	}
}

// --- InitializeKeys ---

func TestJWT_InitializeKeys_Success(t *testing.T) {
	setupInitializedKeys(t)

	if !keysInitialized {
		t.Error("expected keysInitialized = true")
	}
	if cachedPrivateKey == nil {
		t.Error("expected cachedPrivateKey to be set")
	}
	if cachedPublicKey == nil {
		t.Error("expected cachedPublicKey to be set")
	}
	if kid != "test-kid" {
		t.Errorf("kid = %q, want test-kid", kid)
	}
}

func TestJWT_InitializeKeys_AlreadyInitialized_Success(t *testing.T) {
	setupInitializedKeys(t)
	// Second call must be a no-op returning nil.
	if err := InitializeKeys(); err != nil {
		t.Errorf("second InitializeKeys returned error: %v", err)
	}
}

func TestJWT_InitializeKeys_MissingKeys_Failure(t *testing.T) {
	resetJWTState()
	defer resetJWTState()
	os.Unsetenv("JWT_PRIVATE_KEY")
	os.Unsetenv("JWT_PUBLIC_KEY")

	err := InitializeKeys()
	if err == nil {
		t.Error("expected error when JWT keys are not configured")
	}
}

func TestJWT_InitializeKeys_InvalidPrivateKey_Failure(t *testing.T) {
	resetJWTState()
	defer func() {
		os.Unsetenv("JWT_PRIVATE_KEY")
		os.Unsetenv("JWT_PUBLIC_KEY")
		resetJWTState()
	}()
	os.Setenv("JWT_PRIVATE_KEY", "not-valid-pem-data")
	os.Setenv("JWT_PUBLIC_KEY", "not-valid-pem-data")

	err := InitializeKeys()
	if err == nil {
		t.Error("expected error for invalid private key PEM")
	}
}

func TestJWT_InitializeKeys_InvalidPublicKey_Failure(t *testing.T) {
	resetJWTState()
	privPEM, _ := generateTestKeys(t)
	defer func() {
		os.Unsetenv("JWT_PRIVATE_KEY")
		os.Unsetenv("JWT_PUBLIC_KEY")
		resetJWTState()
	}()
	os.Setenv("JWT_PRIVATE_KEY", privPEM)
	os.Setenv("JWT_PUBLIC_KEY", "not-valid-pub-pem")

	err := InitializeKeys()
	if err == nil {
		t.Error("expected error for invalid public key PEM")
	}
}

// --- SignToken ---

func TestJWT_SignToken_Success(t *testing.T) {
	setupInitializedKeys(t)

	token, err := SignToken("test-issuer", "user-123", "test-audience", 3600, []string{"read", "write"})
	if err != nil {
		t.Fatalf("SignToken failed: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token string")
	}
}

func TestJWT_SignToken_NotInitialized_Failure(t *testing.T) {
	resetJWTState()
	defer resetJWTState()

	_, err := SignToken("issuer", "subject", "audience", 3600, nil)
	if err == nil {
		t.Error("expected error when keys are not initialized")
	}
}

// --- ValidateToken ---

func TestJWT_ValidateToken_Success(t *testing.T) {
	setupInitializedKeys(t)

	token, err := SignToken("test-issuer", "user-123", "test-audience", 3600, []string{"read"})
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}

	valid, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if !valid {
		t.Error("expected valid = true for a fresh signed token")
	}
}

func TestJWT_ValidateToken_NotInitialized_Failure(t *testing.T) {
	resetJWTState()
	defer resetJWTState()

	_, err := ValidateToken("any.token.value")
	if err == nil {
		t.Error("expected error when keys are not initialized")
	}
}

func TestJWT_ValidateToken_MissingIssuer_Failure(t *testing.T) {
	setupInitializedKeys(t)
	iss = "" // Clear after initialization.

	_, err := ValidateToken("any.token.value")
	if err == nil {
		t.Error("expected error for missing issuer")
	}
}

func TestJWT_ValidateToken_MissingAudience_Failure(t *testing.T) {
	setupInitializedKeys(t)
	aud = "" // Clear after initialization.

	_, err := ValidateToken("any.token.value")
	if err == nil {
		t.Error("expected error for missing audience")
	}
}

func TestJWT_ValidateToken_InvalidToken_Failure(t *testing.T) {
	setupInitializedKeys(t)

	_, err := ValidateToken("this.is.not.valid")
	if err == nil {
		t.Error("expected error for an invalid token string")
	}
}

func TestJWT_ValidateToken_WrongSigningMethod_Failure(t *testing.T) {
	setupInitializedKeys(t)

	// Sign with HMAC instead of RSA.
	claims := golangJWT.MapClaims{
		"sub": "user",
		"iss": iss,
		"aud": golangJWT.ClaimStrings{aud},
	}
	tok := golangJWT.NewWithClaims(golangJWT.SigningMethodHS256, claims)
	tokenStr, _ := tok.SignedString([]byte("hmac-secret"))

	_, err := ValidateToken(tokenStr)
	if err == nil {
		t.Error("expected error for token signed with wrong method")
	}
}

// --- ParseClaims ---

func TestJWT_ParseClaims_Success(t *testing.T) {
	setupInitializedKeys(t)

	token, err := SignToken("test-issuer", "user-456", "test-audience", 3600, []string{"read", "write"})
	if err != nil {
		t.Fatalf("SignToken: %v", err)
	}

	claims, err := ParseClaims(token)
	if err != nil {
		t.Fatalf("ParseClaims: %v", err)
	}
	if claims.Subject != "user-456" {
		t.Errorf("Subject = %q, want user-456", claims.Subject)
	}
	if len(claims.Scopes) != 2 {
		t.Errorf("Scopes = %v, want [read write]", claims.Scopes)
	}
}

func TestJWT_ParseClaims_NotInitialized_Failure(t *testing.T) {
	resetJWTState()
	defer resetJWTState()

	_, err := ParseClaims("any.token.value")
	if err == nil {
		t.Error("expected error when keys are not initialized")
	}
}

func TestJWT_ParseClaims_InvalidToken_Failure(t *testing.T) {
	setupInitializedKeys(t)

	_, err := ParseClaims("bad.token.string")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestJWT_ParseClaims_WrongSigningMethod_Failure(t *testing.T) {
	setupInitializedKeys(t)

	claims := golangJWT.MapClaims{"sub": "user"}
	tok := golangJWT.NewWithClaims(golangJWT.SigningMethodHS256, claims)
	tokenStr, _ := tok.SignedString([]byte("secret"))

	_, err := ParseClaims(tokenStr)
	if err == nil {
		t.Error("expected error for token with wrong signing method")
	}
}

// --- ExtractTokenFromRequest ---

func TestJWT_ExtractTokenFromRequest_Success(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-test-token")

	token, err := ExtractTokenFromRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "my-test-token" {
		t.Errorf("token = %q, want my-test-token", token)
	}
}

func TestJWT_ExtractTokenFromRequest_NoHeader_Failure(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	_, err := ExtractTokenFromRequest(req)
	if err == nil {
		t.Error("expected error for missing Authorization header")
	}
}

func TestJWT_ExtractTokenFromRequest_EmptyHeader_Failure(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "")

	_, err := ExtractTokenFromRequest(req)
	if err == nil {
		t.Error("expected error for empty Authorization header")
	}
}

func TestJWT_ExtractTokenFromRequest_NonBearerPrefix_Failure(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	_, err := ExtractTokenFromRequest(req)
	if err == nil {
		t.Error("expected error for non-Bearer Authorization header")
	}
}
