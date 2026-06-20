package hashing

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	DefaultMemory      uint32 = 64 * 1024
	DefaultTime        uint32 = 3
	DefaultParallelism uint8  = 2
	DefaultSaltLen     uint32 = 16
	DefaultKeyLen      uint32 = 32
)

type Config struct {
	Memory      uint32
	Time        uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

type Hasher struct {
	memory      uint32
	time        uint32
	parallelism uint8
	saltLen     uint32
	keyLen      uint32
}

func New(cfg Config) (*Hasher, error) {
	if cfg.Memory == 0 {
		cfg.Memory = DefaultMemory
	}
	if cfg.Time == 0 {
		cfg.Time = DefaultTime
	}
	if cfg.Parallelism == 0 {
		cfg.Parallelism = DefaultParallelism
	}
	if cfg.SaltLen == 0 {
		cfg.SaltLen = DefaultSaltLen
	}
	if cfg.KeyLen == 0 {
		cfg.KeyLen = DefaultKeyLen
	}
	if cfg.Memory < 8*uint32(cfg.Parallelism) {
		return nil, fmt.Errorf("requires memory of at least 8 KiB per lane, got %d KiB for %d lanes", cfg.Memory, cfg.Parallelism)
	}
	if cfg.SaltLen < 8 {
		return nil, fmt.Errorf("requires a salt of at least 8 bytes, got %d", cfg.SaltLen)
	}
	if cfg.KeyLen < 16 {
		return nil, fmt.Errorf("requires a key of at least 16 bytes, got %d", cfg.KeyLen)
	}
	return &Hasher{
		memory:      cfg.Memory,
		time:        cfg.Time,
		parallelism: cfg.Parallelism,
		saltLen:     cfg.SaltLen,
		keyLen:      cfg.KeyLen,
	}, nil
}

func (h *Hasher) Hash(plaintext string) (string, error) {
	salt := make([]byte, h.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(plaintext), salt, h.time, h.memory, h.parallelism, h.keyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.memory, h.time, h.parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func Verify(plaintext, encodedHash string) (bool, error) {
	p, salt, key, err := decode(encodedHash)
	if err != nil {
		return false, err
	}
	computed := argon2.IDKey([]byte(plaintext), salt, p.time, p.memory, p.parallelism, uint32(len(key)))
	return subtle.ConstantTimeCompare(computed, key) == 1, nil
}

type params struct {
	memory      uint32
	time        uint32
	parallelism uint8
}

func decode(encodedHash string) (params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" {
		return params{}, nil, nil, errors.New("failed to parse hash: malformed encoding")
	}
	if parts[1] != "argon2id" {
		return params{}, nil, nil, fmt.Errorf("unsupported algorithm %q", parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return params{}, nil, nil, fmt.Errorf("failed to parse hash version: %w", err)
	}
	if version != argon2.Version {
		return params{}, nil, nil, fmt.Errorf("unsupported argon2 version %d", version)
	}

	var mem, tm, par uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &tm, &par); err != nil {
		return params{}, nil, nil, fmt.Errorf("failed to parse hash parameters: %w", err)
	}
	if tm == 0 || par == 0 || par > 255 || mem < 8*par {
		return params{}, nil, nil, errors.New("failed to parse hash: invalid parameters")
	}
	p := params{memory: mem, time: tm, parallelism: uint8(par)}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params{}, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params{}, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	if len(salt) == 0 || len(key) == 0 {
		return params{}, nil, nil, errors.New("failed to parse hash: empty salt or key")
	}
	return p, salt, key, nil
}
