package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type secretsManagerAPI interface {
	GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type API interface {
	GetSecret(ctx context.Context, name string) (string, error)
	GetSecrets(ctx context.Context, keys []string) (map[string]string, error)
}

type Config struct {
	Region     string
	Endpoint   string
	SecretPath string
	Keys       []string
	CacheTTL   time.Duration
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

type parsedEntry struct {
	value     map[string]string
	expiresAt time.Time
}

type secretCache struct {
	mu            sync.RWMutex
	entries       map[string]cacheEntry
	parsedEntries map[string]parsedEntry
	ttl           time.Duration
}

func newCache(ctx context.Context, ttl time.Duration) *secretCache {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	c := &secretCache{
		entries:       make(map[string]cacheEntry),
		parsedEntries: make(map[string]parsedEntry),
		ttl:           ttl,
	}
	go c.evictLoop(ctx)
	return c
}

func (c *secretCache) evictLoop(ctx context.Context) {
	interval := max(c.ttl/2, 30*time.Second)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.evict()
		}
	}
}

func (c *secretCache) evict() {
	now := time.Now()
	c.mu.Lock()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
	for k, e := range c.parsedEntries {
		if now.After(e.expiresAt) {
			delete(c.parsedEntries, k)
		}
	}
	c.mu.Unlock()
}

func (c *secretCache) get(key string) (string, bool) {
	if c.ttl < 0 {
		return "", false
	}
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

func (c *secretCache) set(key, value string) {
	if c.ttl < 0 {
		return
	}
	c.mu.Lock()
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *secretCache) getParsed(key string) (map[string]string, bool) {
	if c.ttl < 0 {
		return nil, false
	}
	c.mu.RLock()
	e, ok := c.parsedEntries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

func (c *secretCache) setParsed(key string, value map[string]string) {
	if c.ttl < 0 {
		return
	}
	c.mu.Lock()
	c.parsedEntries[key] = parsedEntry{value: value, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

type Client struct {
	sm         secretsManagerAPI
	secretPath string
	cache      *secretCache
	sf         singleflight.Group
	cancel     context.CancelFunc
}

func (c *Client) Close() { c.cancel() }

func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("invalid region")
	}

	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}
	if cfg.Endpoint != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var smOpts []func(*secretsmanager.Options)
	if cfg.Endpoint != "" {
		ep := cfg.Endpoint
		smOpts = append(smOpts, func(o *secretsmanager.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	cacheCtx, cancel := context.WithCancel(context.Background())
	c := &Client{
		sm:         secretsmanager.NewFromConfig(awsCfg, smOpts...),
		secretPath: cfg.SecretPath,
		cache:      newCache(cacheCtx, cfg.CacheTTL),
		cancel:     cancel,
	}

	if len(cfg.Keys) > 0 {
		if _, err := c.GetSecrets(ctx, cfg.Keys); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to load initial secrets: %w", err)
		}
	}
	return c, nil
}

func newWithAPI(api secretsManagerAPI, secretPath string) *Client {
	cacheCtx, cancel := context.WithCancel(context.Background())
	return &Client{
		sm:         api,
		secretPath: secretPath,
		cache:      newCache(cacheCtx, 0),
		cancel:     cancel,
	}
}

// Fetches a single secret by its full name; results are cached for CacheTTL.
func (c *Client) GetSecret(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("invalid secret name")
	}

	if v, ok := c.cache.get(name); ok {
		return v, nil
	}

	v, err, _ := c.sf.Do("secret:"+name, func() (any, error) {
		result, err := c.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(name),
		})
		if err != nil {
			return "", err
		}
		if result.SecretString == nil {
			return "", fmt.Errorf("secret has no string value")
		}
		value := *result.SecretString
		c.cache.set(name, value)
		return value, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// Fetches the JSON blob at SecretPath and returns the requested subset of keys; the parsed blob is cached for CacheTTL.
func (c *Client) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("keys must not be empty")
	}
	if c.secretPath == "" {
		return nil, fmt.Errorf("invalid secret path")
	}

	if all, ok := c.cache.getParsed(c.secretPath); ok {
		return extractKeys(all, keys), nil
	}

	raw, err, _ := c.sf.Do("secrets:"+c.secretPath, func() (any, error) {
		result, err := c.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(c.secretPath),
		})
		if err != nil {
			return nil, err
		}
		if result.SecretString == nil {
			return nil, fmt.Errorf("secret has no string value")
		}
		var all map[string]string
		if err := json.Unmarshal([]byte(*result.SecretString), &all); err != nil {
			return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
		}
		c.cache.setParsed(c.secretPath, all)
		return all, nil
	})
	if err != nil {
		return nil, err
	}
	return extractKeys(raw.(map[string]string), keys), nil
}

// Polls SecretPath on a supervised background goroutine, calling onChange only when a
// requested key's value changes; cancel ctx to stop. Close tears down the cache
// eviction loop Watch also relies on, so the client must stay open while watching.
func (c *Client) Watch(ctx context.Context, interval time.Duration, keys []string, onChange func(map[string]string)) {
	if interval <= 0 || len(keys) == 0 || onChange == nil || c.secretPath == "" {
		return
	}
	go c.runWatch(ctx, interval, keys, onChange)
}

func (c *Client) runWatch(ctx context.Context, interval time.Duration, keys []string, onChange func(map[string]string)) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("secret watch loop panicked; restarting", fmt.Errorf("%v", r))
			go c.runWatch(ctx, interval, keys, onChange)
		}
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var previous map[string]string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current, err := c.pollSecrets(ctx, keys)
			if err != nil {
				logger.Error("secret watch poll failed", err)
				continue
			}
			if previous != nil && secretsChanged(previous, current) {
				onChange(current)
			}
			previous = current
		}
	}
}

// Re-fetches directly from Secrets Manager, bypassing the cache, so Watch always observes the live value.
func (c *Client) pollSecrets(ctx context.Context, keys []string) (map[string]string, error) {
	result, err := c.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(c.secretPath),
	})
	if err != nil {
		return nil, err
	}
	if result.SecretString == nil {
		return nil, fmt.Errorf("secret has no string value")
	}
	var all map[string]string
	if err := json.Unmarshal([]byte(*result.SecretString), &all); err != nil {
		return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
	}
	c.cache.setParsed(c.secretPath, all)
	return extractKeys(all, keys), nil
}

func secretsChanged(prev, current map[string]string) bool {
	if len(prev) != len(current) {
		return true
	}
	for k, v := range current {
		if prev[k] != v {
			return true
		}
	}
	return false
}

func extractKeys(all map[string]string, keys []string) map[string]string {
	result := make(map[string]string, len(keys))
	var missing []string
	for _, k := range keys {
		if v, ok := all[k]; ok {
			result[k] = v
		} else {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		logger.Warn("keys not found in secret", "keys", missing)
	}
	return result
}
