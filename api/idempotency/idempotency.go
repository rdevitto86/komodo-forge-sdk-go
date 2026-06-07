package idempotency

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rdevitto86/komodo-forge-sdk-go/db/redis"
)

var evictOnce sync.Once

func startEvictor(c *LocalCache) {
	evictOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				now := time.Now().Unix()
				c.store.Range(func(k, v any) bool {
					if until, ok := v.(int64); ok && until <= now {
						c.store.Delete(k)
					}
					return true
				})
			}
		}()
	})
}

const DEFAULT_IDEM_TTL_SEC int64 = 300 // 5 minutes

type Cache interface {
	Load(key string) (any, bool)
	Store(key string, value any, ttl int64) error
	// stores atomically only when absent, avoiding the race a separate Load-then-Store allows
	StoreIfAbsent(key string, value any, ttl int64) (bool, error)
	Delete(key string)
}

type LocalCache struct {
	store sync.Map
}

func (c *LocalCache) Load(key string) (any, bool) {
	return c.store.Load(key)
}

func (c *LocalCache) Store(key string, value any, ttl int64) error {
	startEvictor(c)
	c.store.Store(key, value)
	return nil
}

// Retries via compare-and-swap to claim a key whose previous entry has expired.
func (c *LocalCache) StoreIfAbsent(key string, value any, ttl int64) (bool, error) {
	startEvictor(c)
	for {
		actual, loaded := c.store.LoadOrStore(key, value)
		if !loaded {
			return true, nil
		}
		if until, ok := actual.(int64); ok && until > time.Now().Unix() {
			return false, nil // live entry — duplicate
		}
		if c.store.CompareAndSwap(key, actual, value) {
			return true, nil // replaced an expired entry
		}
		// lost the race against another writer — retry
	}
}

func (c *LocalCache) Delete(key string) {
	c.store.Delete(key)
}

// relies on Redis's native TTL rather than tracking expiry timestamps like LocalCache does
type DistributedCache struct {
	client redis.API
}

func NewDistributedCache(client redis.API) *DistributedCache {
	return &DistributedCache{client: client}
}

// Reports existence via Exists; the returned value is a sentinel, not a tracked expiry timestamp.
func (c *DistributedCache) Load(key string) (any, bool) {
	exists, err := c.client.Exists(context.Background(), key)
	if err != nil || !exists {
		return nil, false
	}
	return true, true
}

func (c *DistributedCache) Store(key string, value any, ttl int64) error {
	return c.client.Set(context.Background(), key, fmt.Sprint(value), ttl)
}

// Delegates to Redis SETNX for atomic create-if-absent.
func (c *DistributedCache) StoreIfAbsent(key string, value any, ttl int64) (bool, error) {
	return c.client.SetNX(context.Background(), key, fmt.Sprint(value), ttl)
}

func (c *DistributedCache) Delete(key string) {
	_ = c.client.Delete(context.Background(), key)
}

type Store struct {
	cache Cache
	ttl   int64
}

// Creates a Store backed by an in-memory LocalCache; ttl defaults to 300s when 0.
func NewStore(mode string, ttl int64) *Store {
	if ttl == 0 {
		ttl = getIdemTTL()
	}
	return &Store{
		cache: &LocalCache{},
		ttl:   ttl,
	}
}

// Creates a Store backed by a Redis DistributedCache for multi-instance deployments; ttl defaults to 300s when 0.
func NewDistributedStore(client redis.API, ttl int64) *Store {
	if ttl == 0 {
		ttl = getIdemTTL()
	}
	return &Store{
		cache: NewDistributedCache(client),
		ttl:   ttl,
	}
}

// Reports whether key is new, deleting it first if its entry has expired.
func (s *Store) Check(key string) (bool, error) {
	if exp, ok := s.cache.Load(key); ok {
		if until, ok := exp.(int64); ok && until > time.Now().Unix() {
			return false, nil // Key exists and not expired
		}
		s.cache.Delete(key) // Expired, delete it
	}
	return true, nil // Key is new
}

func (s *Store) Set(key string) error {
	expiry := time.Now().Unix() + s.ttl
	return s.cache.Store(key, expiry, s.ttl)
}

// Checks and reserves key atomically; unlike separate Check+Set, two concurrent callers can't both observe "new".
func (s *Store) CheckAndSet(key string) (bool, error) {
	expiry := time.Now().Unix() + s.ttl
	return s.cache.StoreIfAbsent(key, expiry, s.ttl)
}

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
