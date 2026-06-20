package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return key
}

func TestNew_BadKeySize_Failure(t *testing.T) {
	for _, n := range []int{0, 16, 24, 31, 33} {
		if _, err := New(Config{Key: make([]byte, n)}); err == nil {
			t.Errorf("expected error for key size %d", n)
		}
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	c, err := New(Config{Key: testKey(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plaintext := []byte("super-secret-value")
	blob, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := c.Decrypt(blob)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("roundtrip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	c, _ := New(Config{Key: testKey(t)})
	blob, err := c.Encrypt([]byte{})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := c.Decrypt(blob)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty plaintext, got %q", got)
	}
}

func TestDecrypt_Tampered_Failure(t *testing.T) {
	c, _ := New(Config{Key: testKey(t)})
	blob, _ := c.Encrypt([]byte("value"))
	blob[len(blob)-1] ^= 0xFF
	if _, err := c.Decrypt(blob); err == nil {
		t.Error("expected error decrypting tampered ciphertext")
	}
}

func TestDecrypt_WrongKey_Failure(t *testing.T) {
	c1, _ := New(Config{Key: testKey(t)})
	c2, _ := New(Config{Key: testKey(t)})
	blob, _ := c1.Encrypt([]byte("value"))
	if _, err := c2.Decrypt(blob); err == nil {
		t.Error("expected error decrypting with a different key")
	}
}

func TestDecrypt_TooShort_Failure(t *testing.T) {
	c, _ := New(Config{Key: testKey(t)})
	if _, err := c.Decrypt([]byte("x")); err == nil {
		t.Error("expected error for ciphertext shorter than nonce")
	}
}

func TestEncrypt_NonceUniqueness(t *testing.T) {
	c, _ := New(Config{Key: testKey(t)})
	a, _ := c.Encrypt([]byte("same"))
	b, _ := c.Encrypt([]byte("same"))
	if bytes.Equal(a, b) {
		t.Error("expected distinct ciphertexts for repeated encryption")
	}
}
