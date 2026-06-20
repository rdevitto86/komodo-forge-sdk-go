package token_test

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"sync"
	"testing"

	"github.com/rdevitto86/komodo-forge-sdk-go/security/token"
)

func TestBytes_Length(t *testing.T) {
	for _, n := range []int{1, 16, 32, 64} {
		b, err := token.Bytes(n)
		if err != nil {
			t.Fatalf("Bytes(%d): %v", n, err)
		}
		if len(b) != n {
			t.Errorf("Bytes(%d) length = %d", n, len(b))
		}
	}
}

func TestBytes_InvalidLength_Failure(t *testing.T) {
	for _, n := range []int{0, -1, -100} {
		if _, err := token.Bytes(n); err == nil {
			t.Errorf("expected error for Bytes(%d)", n)
		}
	}
}

func TestBytes_Uniqueness(t *testing.T) {
	a, _ := token.Bytes(32)
	b, _ := token.Bytes(32)
	if string(a) == string(b) {
		t.Error("expected distinct random byte slices")
	}
}

func TestHex_FormatAndLength(t *testing.T) {
	s, err := token.Hex(16)
	if err != nil {
		t.Fatalf("Hex: %v", err)
	}
	if len(s) != 32 {
		t.Errorf("Hex(16) length = %d, want 32", len(s))
	}
	if _, err := hex.DecodeString(s); err != nil {
		t.Errorf("Hex output is not valid hex: %v", err)
	}
}

func TestHex_InvalidLength_Failure(t *testing.T) {
	if _, err := token.Hex(0); err == nil {
		t.Error("expected error for Hex(0)")
	}
}

func TestURLSafe_FormatAndDecode(t *testing.T) {
	s, err := token.URLSafe(32)
	if err != nil {
		t.Fatalf("URLSafe: %v", err)
	}
	if strings.ContainsAny(s, "+/=") {
		t.Errorf("URLSafe output %q contains non-url-safe or padding chars", s)
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("URLSafe output not decodable: %v", err)
	}
	if len(b) != 32 {
		t.Errorf("decoded length = %d, want 32", len(b))
	}
}

func TestURLSafe_InvalidLength_Failure(t *testing.T) {
	if _, err := token.URLSafe(-1); err == nil {
		t.Error("expected error for URLSafe(-1)")
	}
}

func TestAPIKey_WithPrefix(t *testing.T) {
	k, err := token.APIKey("sk")
	if err != nil {
		t.Fatalf("APIKey: %v", err)
	}
	if !strings.HasPrefix(k, "sk_") {
		t.Errorf("APIKey = %q, want sk_ prefix", k)
	}
	body := strings.TrimPrefix(k, "sk_")
	if _, err := base64.RawURLEncoding.DecodeString(body); err != nil {
		t.Errorf("APIKey body not url-safe base64: %v", err)
	}
}

func TestAPIKey_EmptyPrefix(t *testing.T) {
	k, err := token.APIKey("")
	if err != nil {
		t.Fatalf("APIKey: %v", err)
	}
	b, err := base64.RawURLEncoding.DecodeString(k)
	if err != nil {
		t.Fatalf("APIKey(\"\") = %q, not url-safe base64: %v", k, err)
	}
	if len(b) != token.DefaultBytes {
		t.Errorf("APIKey(\"\") decoded length = %d, want %d (no prefix should be prepended)", len(b), token.DefaultBytes)
	}
}

func TestAPIKey_Uniqueness(t *testing.T) {
	a, _ := token.APIKey("sk")
	b, _ := token.APIKey("sk")
	if a == b {
		t.Error("expected distinct API keys")
	}
}

func TestConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	seen := sync.Map{}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := token.Hex(16)
			if err != nil {
				t.Errorf("Hex: %v", err)
				return
			}
			if _, dup := seen.LoadOrStore(s, true); dup {
				t.Errorf("duplicate token generated: %s", s)
			}
		}()
	}
	wg.Wait()
}
