package idempotency

import (
	"os"
	"sync"
	"time"
)

const DEFAULT_IDEM_TTL_SEC int64 = 300 // 5 minutes

// Cache defines the interface for idempotency key storage.
// Implementations can be local (in-memory) or distributed (Redis, ElastiCache, etc.).
type Cache interface {
	Load(key string) (interface{}, bool)
	Store(key string, value interface{}, ttl int64) error
	Delete(key string)
}

// LocalCache is an in-memory implementation using sync.Map.
// Suitable for single-instance deployments or testing.
type LocalCache struct {
	store sync.Map
}

func (c *LocalCache) Load(key string) (interface{}, bool) {
	return c.store.Load(key)
}

func (c *LocalCache) Store(key string, value interface{}, ttl int64) error {
	c.store.Store(key, value)
	return nil
}

func (c *LocalCache) Delete(key string) {
	c.store.Delete(key)
}

// DistributedCache is a placeholder for distributed cache implementations
// (Redis, ElastiCache, etc.). Implement when external cache integration is needed.
type DistributedCache struct {
	// client interface for redis/elasticache
}

func (c *DistributedCache) Load(key string) (interface{}, bool) {
	// TODO: Implement distributed cache load
	return nil, false
}

func (c *DistributedCache) Store(key string, value interface{}, ttl int64) error {
	// TODO: Implement distributed cache store
	return nil
}

func (c *DistributedCache) Delete(key string) {
	// TODO: Implement distributed cache delete
}

// Store handles idempotency key operations with a configurable cache backend.
type Store struct {
	cache Cache
	ttl   int64
}

// NewStore creates a new idempotency store.
// mode: "local" for in-memory sync.Map, "distributed" for Redis/ElastiCache
// ttl: time-to-live in seconds (defaults to 300s if 0)
func NewStore(mode string, ttl int64) *Store {
	if ttl == 0 {
		ttl = getIdemTTL()
	}

	var cache Cache
	switch mode {
	case "distributed":
		cache = &DistributedCache{}
	default: // local
		cache = &LocalCache{}
	}

	return &Store{
		cache: cache,
		ttl:   ttl,
	}
}

// Check returns true if the key is new (allowed), false if it already exists (duplicate).
// If the existing key is expired, it is deleted and the key is considered new.
func (s *Store) Check(key string) (bool, error) {
	if exp, ok := s.cache.Load(key); ok {
		if until, ok := exp.(int64); ok && until > time.Now().Unix() {
			return false, nil // Key exists and not expired
		}
		s.cache.Delete(key) // Expired, delete it
	}
	return true, nil // Key is new
}

// Set stores the key with expiration.
func (s *Store) Set(key string) error {
	expiry := time.Now().Unix() + s.ttl
	return s.cache.Store(key, expiry, s.ttl)
}

// Delete removes the key from storage.
func (s *Store) Delete(key string) {
	s.cache.Delete(key)
}

func getIdemTTL() int64 {
	if ttl := os.Getenv("IDEMPOTENCY_TTL_SEC"); ttl != "" {
		if dur, err := time.ParseDuration(ttl + "s"); err == nil {
			if dur <= 0 {
				return DEFAULT_IDEM_TTL_SEC
			}
			return int64(dur.Seconds())
		}
	}
	return DEFAULT_IDEM_TTL_SEC
}