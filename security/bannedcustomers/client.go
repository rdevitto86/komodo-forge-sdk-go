package bannedcustomers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamodb"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

type Checker interface {
	IsBanned(ctx context.Context, email string) (bool, error)
}

type Config struct {
	TableName string
	DynamoDB  dynamodb.API
	FailOpen  *bool
	CacheTTL  time.Duration
}

type Client struct {
	table    string
	db       dynamodb.API
	failOpen bool
	cache    *banCache
}

type record struct {
	Email     string `dynamodbav:"email"`
	ExpiresAt int64  `dynamodbav:"expires_at"`
}

func New(cfg Config) (*Client, error) {
	if cfg.TableName == "" {
		return nil, errors.New("missing table name")
	}
	if cfg.DynamoDB == nil {
		return nil, errors.New("missing dynamodb client")
	}
	failOpen := cfg.FailOpen == nil || *cfg.FailOpen
	c := &Client{table: cfg.TableName, db: cfg.DynamoDB, failOpen: failOpen}
	if cfg.CacheTTL > 0 {
		c.cache = newBanCache(cfg.CacheTTL)
	}
	return c, nil
}

func (c *Client) IsBanned(ctx context.Context, email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false, fmt.Errorf("requires a non-empty email")
	}

	if c.cache != nil {
		if banned, ok := c.cache.get(email); ok {
			return banned, nil
		}
	}

	key, err := c.db.BuildKey("email", email, "", nil)
	if err != nil {
		if c.failOpen {
			logger.Error("failed to build banned-customer lookup key; failing open", err)
			return false, nil
		}
		return false, fmt.Errorf("failed to build banned-customer lookup key: %w", err)
	}

	var rec record
	if err := c.db.GetItemAs(ctx, c.table, key, false, nil, &rec); err != nil {
		if errors.Is(err, dynamodb.ErrNotFound) {
			c.cacheSet(email, false)
			return false, nil
		}
		if c.failOpen {
			logger.Error("banned-customer lookup failed; failing open", err)
			return false, nil
		}
		return false, fmt.Errorf("banned-customer lookup failed: %w", err)
	}

	if rec.ExpiresAt > 0 && rec.ExpiresAt <= time.Now().Unix() {
		c.cacheSet(email, false)
		return false, nil
	}
	c.cacheSet(email, true)
	return true, nil
}

func (c *Client) cacheSet(email string, banned bool) {
	if c.cache != nil {
		c.cache.set(email, banned)
	}
}

type cacheEntry struct {
	banned    bool
	expiresAt time.Time
}

type banCache struct {
	ttl time.Duration
	mu  sync.Mutex
	m   map[string]cacheEntry
}

func newBanCache(ttl time.Duration) *banCache {
	return &banCache{ttl: ttl, m: make(map[string]cacheEntry)}
}

func (b *banCache) get(key string) (bool, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.m[key]
	if !ok {
		return false, false
	}
	if time.Now().After(e.expiresAt) {
		delete(b.m, key)
		return false, false
	}
	return e.banned, true
}

func (b *banCache) set(key string, banned bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.m[key] = cacheEntry{banned: banned, expiresAt: time.Now().Add(b.ttl)}
}
