package jwt

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	cachedPrivateKey *rsa.PrivateKey
	cachedPublicKey  *rsa.PublicKey
	kid              string
	iss              string
	aud              string
	keyMutex         sync.RWMutex
	keysInitialized  atomic.Bool
)

type CustomClaims struct {
	Scopes  []string `json:"scp,omitempty"`
	IsAdmin bool     `json:"adm,omitempty"`
	jwt.RegisteredClaims
}

// Loads RSA signing and verification keys from environment variables and assigns a KID for rotation support.
func InitializeKeys() error {
	if keysInitialized.Load() {
		return nil
	}

	keyMutex.Lock()
	defer keyMutex.Unlock()

	// Re-check under the write lock: a concurrent first caller may have loaded the
	// keys between the unlocked Load above and acquiring the lock here.
	if keysInitialized.Load() {
		return nil
	}

	kid = os.Getenv("JWT_KID")
	iss = os.Getenv("JWT_ISSUER")
	aud = os.Getenv("JWT_AUDIENCE")
	privKey := os.Getenv("JWT_PRIVATE_KEY")
	pubKey := os.Getenv("JWT_PUBLIC_KEY")

	if privKey == "" || pubKey == "" {
		return fmt.Errorf("JWT keys not fully configured in environment")
	}

	var err error
	cachedPrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(privKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	cachedPublicKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(pubKey))
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	keysInitialized.Store(true)
	return nil
}

// Mints a signed JWT with the given claims and a KID header for key rotation.
func SignToken(issuer string, subject string, audience string, ttl int64, scopes []string) (string, error) {
	if !keysInitialized.Load() {
		return "", fmt.Errorf("failed to sign token: jwt keys not initialized")
	}

	keyMutex.RLock()
	defer keyMutex.RUnlock()

	claims := CustomClaims{
		Scopes: scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(cachedPrivateKey)
}

// Validates a token's signature, expiration, issuer, and audience against env-configured values.
func ValidateToken(tokenString string) (bool, error) {
	if !keysInitialized.Load() {
		return false, fmt.Errorf("failed to validate token: jwt keys not initialized")
	}

	keyMutex.RLock()
	pub := cachedPublicKey
	keyMutex.RUnlock()

	if iss == "" {
		return false, fmt.Errorf("missing jwt issuer")
	}
	if aud == "" {
		return false, fmt.Errorf("missing jwt audience")
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return pub, nil
		},
		jwt.WithIssuer(iss),
		jwt.WithAudience(aud),
		jwt.WithValidMethods([]string{"RS256"}),
	)

	if err != nil {
		return false, fmt.Errorf("verification failed: %w", err)
	}
	if !token.Valid {
		return false, fmt.Errorf("invalid token")
	}

	// If the JTI is in Redis, it means this token was revoked/logged out
	// if isBlacklisted(context.Background(), claims.ID) {
	// 	return false, fmt.Errorf("token has been revoked")
	// }

	return true, nil
}

// Validates the token and returns its embedded claims in a single parse; prefer over ValidateToken + ParseClaims.
func ValidateAndParseClaims(tokenString string) (*CustomClaims, error) {
	if !keysInitialized.Load() {
		return nil, fmt.Errorf("failed to validate token: jwt keys not initialized")
	}

	keyMutex.RLock()
	pub := cachedPublicKey
	keyMutex.RUnlock()

	if iss == "" {
		return nil, fmt.Errorf("missing jwt issuer")
	}
	if aud == "" {
		return nil, fmt.Errorf("missing jwt audience")
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return pub, nil
		},
		jwt.WithIssuer(iss),
		jwt.WithAudience(aud),
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	return claims, nil
}

// Parses and returns the embedded claims from a token string without re-validating issuer or audience.
func ParseClaims(tokenString string) (*CustomClaims, error) {
	if !keysInitialized.Load() {
		return nil, fmt.Errorf("failed to parse claims: jwt keys not initialized")
	}

	keyMutex.RLock()
	pub := cachedPublicKey
	keyMutex.RUnlock()

	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return pub, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	return claims, nil
}

// Extracts the Bearer token string from the Authorization header.
func ExtractTokenFromRequest(req *http.Request) (string, error) {
	auth := req.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return "", fmt.Errorf("missing or invalid authorization header")
	}
	return strings.TrimPrefix(auth, "Bearer "), nil
}
