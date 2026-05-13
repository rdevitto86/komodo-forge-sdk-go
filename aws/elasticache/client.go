package elasticache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/redis/go-redis/v9"
)

type API interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl int64) error
	Delete(ctx context.Context, key string) error
	Close() error
	AllowDistributed(ctx context.Context, key string, rate, burst float64, ttlSec int) (bool, time.Duration, error)
}

type Config struct {
	Endpoint string
	Password string
	DB       int
}

type Client struct {
	rc *redis.Client
}

// Creates a Client and pings the endpoint to verify connectivity.
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	rc := redis.NewClient(&redis.Options{
		Addr:     cfg.Endpoint,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		logger.Error("failed to ping elasticache", err)
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	logger.Info("elasticache client initialized")
	return &Client{rc: rc}, nil
}

// Creates a Client from a Redis connection URL (redis://:password@host:port/db).
func NewFromString(connStr string) (*Client, error) {
	opts, err := redis.ParseURL(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	rc := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		logger.Error("failed to ping elasticache", err)
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	logger.Info("elasticache client initialized from connection string")
	return &Client{rc: rc}, nil
}

// Creates a Client from discrete string fields for callers
// still carrying a legacy string-typed DB value.
func NewFromDBString(endpoint, password, dbStr string) (*Client, error) {
	db, _ := strconv.Atoi(dbStr)
	return New(Config{Endpoint: endpoint, Password: password, DB: db})
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 2*time.Second)
}

// Retrieves the string value stored at key. Returns ("", nil) on cache miss.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	val, err := c.rc.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		logger.Error("failed to get cache item", err)
		return "", err
	}
	return val, nil
}

// Stores a value with the given TTL in seconds. Use ttl=0 for no expiration.
func (c *Client) Set(ctx context.Context, key string, value string, ttl int64) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var dur time.Duration
	if ttl > 0 {
		dur = time.Duration(ttl) * time.Second
	}

	if err := c.rc.Set(ctx, key, value, dur).Err(); err != nil {
		logger.Error("failed to set cache item", err)
		return err
	}
	return nil
}

// Removes a key from the cache.
func (c *Client) Delete(ctx context.Context, key string) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	if err := c.rc.Del(ctx, key).Err(); err != nil {
		logger.Error("failed to delete cache item", err)
		return err
	}
	return nil
}

// Closes the underlying Redis connection.
func (c *Client) Close() error {
	return c.rc.Close()
}

// token bucket Lua script (atomic): returns {allowed, wait_ms}
var tokenBucketScript = redis.NewScript(`
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local burst = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local data = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
if tokens == nil then
  tokens = burst
  ts = now
end
local elapsed = (now - ts) / 1000.0
if elapsed < 0 then elapsed = 0 end
local new_tokens = tokens + elapsed * rate
if new_tokens > burst then new_tokens = burst end
local allowed = 0
local wait_ms = 0
if new_tokens >= requested then
  new_tokens = new_tokens - requested
  allowed = 1
else
  local deficit = requested - new_tokens
  if rate > 0 then
	wait_ms = math.ceil((deficit / rate) * 1000)
  else
	wait_ms = 0
  end
end
redis.call('HMSET', KEYS[1], 'tokens', tostring(new_tokens), 'ts', tostring(now))
redis.call('EXPIRE', KEYS[1], ttl)
return {allowed, tostring(wait_ms)}
`)

// Attempts to consume a token from a distributed token bucket.
// Returns (allowed, retryAfter, error).
func (c *Client) AllowDistributed(ctx context.Context, key string, rate, burst float64, ttlSec int) (bool, time.Duration, error) {
	now := time.Now().UnixMilli()
	res, err := tokenBucketScript.Run(ctx, c.rc, []string{key}, now, rate, burst, 1, ttlSec).Result()
	if err != nil {
		logger.Error("failed to execute token bucket script", err)
		return false, 0, err
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return false, 0, fmt.Errorf("unexpected script result: %v", res)
	}

	var allowed bool
	switch v := arr[0].(type) {
	case int64:
		allowed = v == 1
	case string:
		allowed = v == "1"
	}

	var waitMs int64
	switch v := arr[1].(type) {
	case int64:
		waitMs = v
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			waitMs = parsed
		}
	}

	return allowed, time.Duration(waitMs) * time.Millisecond, nil
}
