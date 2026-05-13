package aurora

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

// Client wraps a database/sql connection pool for Aurora.
// TODO: wire database/sql + pgx or mysql driver when implementing.
type Client struct{}

// Creates and returns a new Aurora Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	panic("aurora: not yet implemented")
}

func (c *Client) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	panic("aurora: not yet implemented")
}
