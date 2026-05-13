package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// API is the interface for Secrets Manager operations. Use it in callers so the
// concrete *Client can be swapped for a test double without modifying call sites.
type API interface {
	GetSecret(key, prefix string) (string, error)
	GetSecrets(keys []string, prefix, batchID string) (map[string]string, error)
}

type Config struct {
	Region   string
	Endpoint string
	Prefix   string
	Batch    string
	Keys     []string
	// CacheTTL is how long secrets are cached in memory before re-fetching.
	// Defaults to 5 minutes when 0. Set to a negative value to disable caching.
	CacheTTL time.Duration
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

type secretCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

func newCache(ttl time.Duration) *secretCache {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &secretCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// get returns the cached value and true if the entry exists and has not expired.
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

// set stores a value in the cache with the configured TTL.
func (c *secretCache) set(key, value string) {
	if c.ttl < 0 {
		return
	}
	c.mu.Lock()
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

// Client wraps the AWS Secrets Manager SDK client.
type Client struct {
	sm     *secretsmanager.Client
	prefix string
	cache  *secretCache
}

// New creates a Secrets Manager Client. If cfg.Keys is non-empty it eagerly
// loads those secrets via GetSecrets and returns any retrieval error.
func New(cfg Config) (*Client, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("secretsmanager: region is required")
	}

	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}
	if cfg.Endpoint != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		logger.Error("failed to load AWS config", err)
		return nil, err
	}

	var smOpts []func(*secretsmanager.Options)
	if cfg.Endpoint != "" {
		ep := cfg.Endpoint
		smOpts = append(smOpts, func(o *secretsmanager.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	c := &Client{
		sm:     secretsmanager.NewFromConfig(awsCfg, smOpts...),
		prefix: cfg.Prefix,
		cache:  newCache(cfg.CacheTTL),
	}

	if len(cfg.Keys) > 0 {
		if _, err := c.GetSecrets(cfg.Keys, cfg.Prefix, cfg.Batch); err != nil {
			logger.Error("failed to load secrets", err)
			return nil, err
		}
	}
	return c, nil
}

// GetSecret retrieves a single secret by key under the given prefix path.
// Results are cached for the configured CacheTTL.
func (c *Client) GetSecret(key, prefix string) (string, error) {
	if prefix == "" {
		return "", fmt.Errorf("secretsmanager: prefix is required")
	}

	path := prefix + key
	if v, ok := c.cache.get(path); ok {
		return v, nil
	}

	result, err := c.sm.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(path),
	})
	if err != nil {
		logger.Error(fmt.Sprintf("failed to retrieve secret %s", path), err)
		return "", err
	}
	if result.SecretString == nil {
		return "", fmt.Errorf("secretsmanager: secret %s has no string value", path)
	}

	value := *result.SecretString
	c.cache.set(path, value)
	logger.Info(fmt.Sprintf("retrieved secret %s", key))
	return value, nil
}

// GetSecrets retrieves a batch secret JSON blob at prefix+batchID and returns
// the subset of keys requested. The full blob is cached for CacheTTL; individual
// key lookups served from that cached blob do not incur additional API calls.
func (c *Client) GetSecrets(keys []string, prefix, batchID string) (map[string]string, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("secretsmanager: no keys provided")
	}
	if prefix == "" {
		return nil, fmt.Errorf("secretsmanager: prefix is required")
	}
	if batchID == "" {
		return nil, fmt.Errorf("secretsmanager: batchID is required")
	}

	path := prefix + batchID

	var raw string
	if v, ok := c.cache.get(path); ok {
		raw = v
	} else {
		result, err := c.sm.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(path),
		})
		if err != nil {
			logger.Error(fmt.Sprintf("failed to retrieve batch secret %s", path), err)
			return nil, err
		}
		if result.SecretString == nil {
			return nil, fmt.Errorf("secretsmanager: batch secret %s has no string value", path)
		}
		raw = *result.SecretString
		c.cache.set(path, raw)
	}

	var all map[string]string
	if err := json.Unmarshal([]byte(raw), &all); err != nil {
		logger.Error("failed to parse batch secret "+path, err)
		return nil, err
	}

	secrets := make(map[string]string, len(keys))
	var missing []string
	for _, k := range keys {
		if v, ok := all[k]; ok {
			secrets[k] = v
		} else {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		logger.Warn(fmt.Sprintf("keys not found in batch secret: %v", missing))
	}

	logger.Info(fmt.Sprintf("retrieved %d secrets from batch", len(secrets)))
	return secrets, nil
}
