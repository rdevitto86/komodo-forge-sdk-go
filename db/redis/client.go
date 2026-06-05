package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	goredis "github.com/redis/go-redis/v9"
)

type API interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl int64) error
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl int64) error
	SetNX(ctx context.Context, key string, value string, ttl int64) (bool, error)
	Exists(ctx context.Context, key string) (bool, error)
	Close() error
	AllowDistributed(ctx context.Context, key string, rate, burst float64, ttlSec int) (bool, time.Duration, error)
}

type Config struct {
	Addr        string
	Password    string
	DB          int
	DialTimeout time.Duration // defaults to 3s when zero
	OpTimeout   time.Duration // defaults to 2s when zero
}

type Client struct {
	rc        *goredis.Client
	opTimeout time.Duration
}

func New(cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("missing addr")
	}

	dialTimeout := cfg.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 3 * time.Second
	}

	rc := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		logger.Error("failed to ping redis", err)
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	logger.Info("redis client initialized")
	return &Client{rc: rc, opTimeout: cfg.OpTimeout}, nil
}

func NewFromString(connStr string) (*Client, error) {
	opts, err := goredis.ParseURL(connStr)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	rc := goredis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		logger.Error("failed to ping redis", err)
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	logger.Info("redis client initialized from connection string")
	return &Client{rc: rc, opTimeout: 0}, nil
}

func NewFromDBString(addr, password, dbStr string) (*Client, error) {
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return nil, fmt.Errorf("invalid db string %q: %w", dbStr, err)
	}
	return New(Config{Addr: addr, Password: password, DB: db})
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	d := c.opTimeout
	if d == 0 {
		d = 2 * time.Second
	}
	return context.WithTimeout(ctx, d)
}

// Retrieves the string value stored at key. Returns ("", nil) on cache miss.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	val, err := c.rc.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", nil
	}
	if err != nil {
		logger.Error("failed to get redis key", err)
		return "", err
	}
	return val, nil
}

// Stores a value with the given TTL in seconds. Use ttl=0 for no expiration.
func (c *Client) Set(ctx context.Context, key string, value string, ttl int64) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	var dur time.Duration
	if ttl > 0 {
		dur = time.Duration(ttl) * time.Second
	}

	if err := c.rc.Set(ctx, key, value, dur).Err(); err != nil {
		logger.Error("failed to set redis key", err)
		return err
	}
	return nil
}

// Atomically increments the integer value at key by one and returns the new value.
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	val, err := c.rc.Incr(ctx, key).Result()
	if err != nil {
		logger.Error("failed to increment redis key", err)
		return 0, err
	}
	return val, nil
}

// Sets key to value only if the key does not already exist; returns true if the write occurred.
func (c *Client) SetNX(ctx context.Context, key string, value string, ttl int64) (bool, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	var dur time.Duration
	if ttl > 0 {
		dur = time.Duration(ttl) * time.Second
	}

	ok, err := c.rc.SetNX(ctx, key, value, dur).Result()
	if err != nil {
		logger.Error("failed to set redis key if not exists", err)
		return false, err
	}
	return ok, nil
}

// Reports whether a key exists in Redis.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	n, err := c.rc.Exists(ctx, key).Result()
	if err != nil {
		logger.Error("failed to check redis key existence", err)
		return false, err
	}
	return n > 0, nil
}

// Removes a key from Redis.
func (c *Client) Delete(ctx context.Context, key string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	if err := c.rc.Del(ctx, key).Err(); err != nil {
		logger.Error("failed to delete redis key", err)
		return err
	}
	return nil
}

// Sets the expiry on an existing key to ttl seconds. A ttl of zero or
// negative is a no-op.
func (c *Client) Expire(ctx context.Context, key string, ttl int64) error {
	if ttl <= 0 {
		return nil
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	if err := c.rc.Expire(ctx, key, time.Duration(ttl)*time.Second).Err(); err != nil {
		logger.Error("failed to set redis key expiry", err)
		return err
	}
	return nil
}

func (c *Client) Close() error {
	return c.rc.Close()
}

// atomic token bucket; returns {allowed, wait_ms}
var tokenBucketScript = goredis.NewScript(`
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

// Attempts to consume a token from a distributed token bucket; returns whether the request is allowed and how long to wait if not.
func (c *Client) AllowDistributed(ctx context.Context, key string, rate, burst float64, ttlSec int) (bool, time.Duration, error) {
	now := time.Now().UnixMilli()
	res, err := tokenBucketScript.Run(ctx, c.rc, []string{key}, now, rate, burst, 1, ttlSec).Result()
	if err != nil {
		logger.Error("failed to execute token bucket script", err)
		return false, 0, err
	}

	arr, ok := res.([]any)
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
