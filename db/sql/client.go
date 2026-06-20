package sqldb

import "context"

type API interface {
	Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error)
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
}

type Config struct {
	DSN         string
	MaxOpenConn int
	MaxIdleConn int
}

type Client struct{}

func New(config Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	return 0, ErrNotImplemented
}
