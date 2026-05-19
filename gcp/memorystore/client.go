package memorystore

// GCP Memorystore (Redis) client — equivalent to aws/elasticache.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import (
	"context"
	"time"
)

type API interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type Config struct {
	Address      string // host:port
	Password     string
	DB           int
	UseTLS       bool
	IAMAuth      bool
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Get(_ context.Context, _ string) (string, error) {
	panic("memorystore: not implemented")
}
func (c *Client) Set(_ context.Context, _, _ string, _ time.Duration) error {
	panic("memorystore: not implemented")
}
func (c *Client) Delete(_ context.Context, _ string) error {
	panic("memorystore: not implemented")
}
func (c *Client) Exists(_ context.Context, _ string) (bool, error) {
	panic("memorystore: not implemented")
}
