package ratelimit

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rdevitto86/komodo-forge-sdk-go/db/redis"
)

type bucket struct {
	mu      sync.Mutex
	tokens  float64
	last    time.Time
	created time.Time
}

type Service interface {
	Allow(ctx context.Context, key string) (allowed bool, wait time.Duration, err error)
	GetUsage(ctx context.Context, key string) (used int, remaining int, reset time.Time, err error)
	Reset(ctx context.Context, key string) error
	LoadConfig(cfg Config) error
	ShouldFailOpen() bool
}

type Config struct {
	RPS             float64
	Burst           float64
	BucketTTLSecond int
	FailOpen        *bool
}

// Holds rate-limit parameters stored in an atomic pointer so LoadConfig and rateConfig are race-free.
type rlCfg struct {
	rps   float64
	burst float64
}

type envCfg struct {
	env    string
	ttlSec int
}

var (
	cfgPtr  atomic.Pointer[rlCfg]
	cfgOnce sync.Once

	envPtr  atomic.Pointer[envCfg]
	envOnce sync.Once

	buckets        sync.Map
	evictOnce      sync.Once
	redisClientVal atomic.Value // stores *redisHolder; use loadRedis() to read

	// stored as an atomic int32 (0 = not loaded, 1 = open, 2 = closed);
	// separate from envCfg so ShouldFailOpen stays cheap
	failOpenVal int32
	failOnce    sync.Once
)

// Wraps the redis.API interface so atomic.Value can store it.
type redisHolder struct{ c redis.API }

// Returns the current Redis client, or nil if none is set.
func loadRedis() redis.API {
	if h, ok := redisClientVal.Load().(*redisHolder); ok {
		return h.c
	}
	return nil
}

// Wires a distributed Redis client for prod/staging rate limiting; must be called before the first Allow call.
func SetRedisClient(c redis.API) {
	redisClientVal.Store(&redisHolder{c: c})
}

// Attempts to consume a token for the given client key.
func Allow(ctx context.Context, key string) (allowed bool, wait time.Duration, err error) {
	env := loadEnv().env

	if env == "prod" || env == "staging" {
		rc := loadRedis()
		if rc == nil {
			return false, 0, fmt.Errorf("no distributed client configured for env %q", env)
		}
		cfg := loadCfg()
		ttlSec := loadEnv().ttlSec
		return rc.AllowDistributed(ctx, key, cfg.rps, cfg.burst, ttlSec)
	}

	b := getBucket(key)
	if ok, wait := b.allow(); !ok {
		return false, wait, nil
	}
	return true, 0, nil
}

// Returns simple usage metrics for the given key.
func GetUsage(ctx context.Context, key string) (used int, remaining int, reset time.Time, err error) {
	bkt := getBucket(key)

	bkt.mu.Lock()
	tokens := bkt.tokens
	bkt.mu.Unlock()

	cfg := loadCfg()
	usedF := cfg.burst - tokens
	if usedF < 0 {
		usedF = 0
	}

	return int(usedF), int(tokens), time.Now().Add(bkt.retryAfter()), nil
}

// Removes any in-process bucket state for the given key.
func Reset(ctx context.Context, key string) error {
	buckets.Delete(key)
	return nil
}

// Programmatically overrides rate limiter settings (RPS/Burst).
func LoadConfig(cfg Config) error {
	current := loadCfg()
	next := rlCfg{rps: current.rps, burst: current.burst}
	if cfg.RPS > 0 {
		next.rps = cfg.RPS
	}
	if cfg.Burst > 0 {
		next.burst = cfg.Burst
	}
	cfgPtr.Store(&next)
	return nil
}

func (bkt *bucket) allow() (bool, time.Duration) {
	cfg := loadCfg()
	now := time.Now()
	bkt.mu.Lock()
	defer bkt.mu.Unlock()

	if !bkt.last.IsZero() {
		elapsed := now.Sub(bkt.last).Seconds()
		if elapsed > 0 {
			bkt.tokens += elapsed * cfg.rps
			if bkt.tokens > cfg.burst {
				bkt.tokens = cfg.burst
			}
		}
	} else {
		bkt.tokens = cfg.burst
	}

	if bkt.tokens >= 1 {
		bkt.tokens--
		bkt.last = now
		return true, 0
	}
	bkt.last = now

	// compute wait while still holding the lock, avoiding a second lock round-trip
	deficit := 1 - bkt.tokens
	if deficit <= 0 || cfg.rps <= 0 {
		return false, time.Second
	}
	return false, time.Duration(deficit / cfg.rps * float64(time.Second))
}

func (bkt *bucket) retryAfter() time.Duration {
	cfg := loadCfg()
	if cfg.rps <= 0 {
		return time.Second
	}

	bkt.mu.Lock()
	defer bkt.mu.Unlock()

	deficit := 1 - bkt.tokens
	if deficit <= 0 {
		return 0
	}
	return time.Duration(deficit / cfg.rps * float64(time.Second))
}

func loadCfg() rlCfg {
	cfgOnce.Do(func() {
		// Don't clobber a config already set explicitly via LoadConfig.
		if cfgPtr.Load() != nil {
			return
		}
		parseFloat := func(key string, dflt float64) float64 {
			if val := strings.TrimSpace(os.Getenv(key)); val != "" {
				if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
					return f
				}
			}
			return dflt
		}
		rps := parseFloat("RATE_LIMIT_RPS", 10)
		burst := parseFloat("RATE_LIMIT_BURST", 20)
		if burst < 1 {
			burst = 20
		}
		cfgPtr.Store(&rlCfg{rps: rps, burst: burst})
	})

	if p := cfgPtr.Load(); p != nil {
		return *p
	}
	return rlCfg{rps: 10, burst: 20}
}

func rateConfig() (float64, float64) {
	c := loadCfg()
	return c.rps, c.burst
}

// Resets all package-level state so tests can start clean; must only be called from test code.
func ResetForTesting() {
	cfgOnce = sync.Once{}
	envOnce = sync.Once{}
	evictOnce = sync.Once{}
	failOnce = sync.Once{}
	cfgPtr.Store(nil)
	envPtr.Store(nil)
	atomic.StoreInt32(&failOpenVal, 0)
	buckets.Range(func(k, v any) bool { buckets.Delete(k); return true })
}

func loadEnv() envCfg {
	envOnce.Do(func() {
		env := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
		ttlSec := 0
		if val := strings.TrimSpace(os.Getenv("RATE_LIMIT_BUCKET_TTL_SEC")); val != "" {
			if i, err := strconv.Atoi(val); err == nil {
				ttlSec = i
			}
		}
		envPtr.Store(&envCfg{env: env, ttlSec: ttlSec})
	})
	return *envPtr.Load()
}

func getBucket(key string) *bucket {
	evictOnce.Do(startBucketEvictor)
	if v, ok := buckets.Load(key); ok {
		return v.(*bucket)
	}
	bkt := &bucket{tokens: 0, last: time.Time{}, created: time.Now()}
	actual, _ := buckets.LoadOrStore(key, bkt)
	return actual.(*bucket)
}

func startBucketEvictor() {
	// sourced from envCfg (RATE_LIMIT_BUCKET_TTL_SEC); falls back to 300s when unset
	ttlSec := loadEnv().ttlSec
	if ttlSec <= 0 {
		ttlSec = 300
	}
	ttl := time.Duration(ttlSec) * time.Second
	ticker := time.NewTicker(time.Minute)

	go func() {
		for range ticker.C {
			now := time.Now()
			buckets.Range(func(key, val any) bool {
				b := val.(*bucket)

				b.mu.Lock()
				lastActive := b.last
				if lastActive.IsZero() {
					lastActive = b.created
				}
				b.mu.Unlock()

				if now.Sub(lastActive) > ttl {
					buckets.Delete(key)
				}
				return true
			})
		}
	}()
}

// Decides fail-open vs fail-closed when the distributed store is unavailable.
func ShouldFailOpen() bool {
	failOnce.Do(func() {
		v := strings.ToLower(strings.TrimSpace(os.Getenv("RATE_LIMIT_FAIL_OPEN")))
		open := v == "" || v == "true" || v == "1" || v == "yes"
		if open {
			atomic.StoreInt32(&failOpenVal, 1)
		} else {
			atomic.StoreInt32(&failOpenVal, 2)
		}
	})
	return atomic.LoadInt32(&failOpenVal) == 1
}
