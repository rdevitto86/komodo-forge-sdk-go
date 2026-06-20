package token

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const DefaultBytes = 32

func Bytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("requires a positive byte length, got %d", n)
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}
	return b, nil
}

func Hex(n int) (string, error) {
	b, err := Bytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func URLSafe(n int) (string, error) {
	b, err := Bytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func APIKey(prefix string) (string, error) {
	s, err := URLSafe(DefaultBytes)
	if err != nil {
		return "", err
	}
	if prefix == "" {
		return s, nil
	}
	return prefix + "_" + s, nil
}
