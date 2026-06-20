package hashing

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/argon2"
)

func fastHasher(t *testing.T) *Hasher {
	t.Helper()
	h, err := New(Config{Memory: 8 * 1024, Time: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h
}

func TestNew_DefaultsApplied(t *testing.T) {
	h, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if h.memory != DefaultMemory || h.time != DefaultTime || h.parallelism != DefaultParallelism {
		t.Errorf("defaults not applied: %+v", h)
	}
	if h.saltLen != DefaultSaltLen || h.keyLen != DefaultKeyLen {
		t.Errorf("salt/key defaults not applied: %+v", h)
	}
}

func TestNew_InvalidParams_Failure(t *testing.T) {
	cases := []Config{
		{Memory: 4, Parallelism: 1},
		{SaltLen: 4},
		{KeyLen: 8},
	}
	for i, cfg := range cases {
		if _, err := New(cfg); err == nil {
			t.Errorf("case %d: expected error for %+v", i, cfg)
		}
	}
}

func TestHashVerify_Roundtrip(t *testing.T) {
	h := fastHasher(t)
	encoded, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	ok, err := Verify("correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Error("expected matching password to verify")
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	h := fastHasher(t)
	encoded, _ := h.Hash("right-password")
	ok, err := Verify("wrong-password", encoded)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Error("expected wrong password to fail verification")
	}
}

func TestVerify_DefaultHasherRoundtrip(t *testing.T) {
	h, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	encoded, err := h.Hash("s3cret")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	ok, err := Verify("s3cret", encoded)
	if err != nil || !ok {
		t.Fatalf("Verify = %v, %v; want true, nil", ok, err)
	}
}

func TestHash_SaltUniqueness(t *testing.T) {
	h := fastHasher(t)
	a, _ := h.Hash("same")
	b, _ := h.Hash("same")
	if a == b {
		t.Error("expected distinct hashes for repeated input (random salt)")
	}
}

func TestHash_EncodingShape(t *testing.T) {
	h := fastHasher(t)
	encoded, _ := h.Hash("x")
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" {
		t.Fatalf("unexpected encoding %q", encoded)
	}
	if parts[1] != "argon2id" {
		t.Errorf("algorithm = %q, want argon2id", parts[1])
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
		t.Errorf("version segment = %q", parts[2])
	}
}

func TestVerify_Malformed_Failure(t *testing.T) {
	h := fastHasher(t)
	good, _ := h.Hash("pw")

	cases := map[string]string{
		"empty":            "",
		"too few fields":   "$argon2id$v=19$m=8192,t=1,p=1$onlysalt",
		"wrong algorithm":  strings.Replace(good, "argon2id", "argon2i", 1),
		"bad version":      strings.Replace(good, fmt.Sprintf("v=%d", argon2.Version), "v=999", 1),
		"bad salt base64":  "$argon2id$v=19$m=8192,t=1,p=1$!!!!$" + strings.Split(good, "$")[5],
		"bad key base64":   "$argon2id$v=19$m=8192,t=1,p=1$" + strings.Split(good, "$")[4] + "$!!!!",
		"bad params":       "$argon2id$v=19$m=garbage$" + strings.Split(good, "$")[4] + "$" + strings.Split(good, "$")[5],
		"zero parallelism": "$argon2id$v=19$m=8192,t=1,p=0$" + strings.Split(good, "$")[4] + "$" + strings.Split(good, "$")[5],
	}
	for name, enc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Verify("pw", enc); err == nil {
				t.Errorf("expected error for %s", name)
			}
		})
	}
}

func TestVerify_TamperedKey(t *testing.T) {
	h := fastHasher(t)
	good, _ := h.Hash("pw")
	parts := strings.Split(good, "$")
	decoded, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	decoded[0] ^= 0x01
	parts[5] = base64.RawStdEncoding.EncodeToString(decoded)
	ok, err := Verify("pw", strings.Join(parts, "$"))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Error("expected tampered key to fail verification with a clean mismatch")
	}
}

func TestHashVerify_EmptyPlaintext(t *testing.T) {
	h := fastHasher(t)
	encoded, err := h.Hash("")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	ok, err := Verify("", encoded)
	if err != nil || !ok {
		t.Fatalf("Verify empty = %v, %v; want true, nil", ok, err)
	}
	if ok, _ := Verify("x", encoded); ok {
		t.Error("expected non-empty password to fail against an empty-password hash")
	}
}

func TestVerify_UsesStoredParams(t *testing.T) {
	h1, err := New(Config{Memory: 8 * 1024, Time: 1, Parallelism: 1})
	if err != nil {
		t.Fatalf("New h1: %v", err)
	}
	h2, err := New(Config{Memory: 16 * 1024, Time: 2, Parallelism: 2})
	if err != nil {
		t.Fatalf("New h2: %v", err)
	}
	e1, _ := h1.Hash("pw")
	e2, _ := h2.Hash("pw")
	if !strings.Contains(e1, "m=8192,t=1,p=1") {
		t.Fatalf("e1 params unexpected: %q", e1)
	}
	if !strings.Contains(e2, "m=16384,t=2,p=2") {
		t.Fatalf("e2 params unexpected: %q", e2)
	}
	for _, enc := range []string{e1, e2} {
		ok, err := Verify("pw", enc)
		if err != nil || !ok {
			t.Errorf("Verify with stored params failed for %q: %v, %v", enc, ok, err)
		}
	}
}

func TestHashVerify_Concurrent(t *testing.T) {
	h := fastHasher(t)
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			pw := fmt.Sprintf("password-%d", n)
			encoded, err := h.Hash(pw)
			if err != nil {
				t.Errorf("Hash: %v", err)
				return
			}
			ok, err := Verify(pw, encoded)
			if err != nil || !ok {
				t.Errorf("Verify = %v, %v; want true, nil", ok, err)
			}
		}(i)
	}
	wg.Wait()
}
