package sqldb

import "context"

// Defines the SQL operations provided by this package.
type API interface {
	Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error)
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
}

// Holds connection parameters for a SQL client.
type Config struct {
	DSN         string
	MaxOpenConn int
	MaxIdleConn int
}

// Wraps a database/sql connection pool for a driver-agnostic SQL database.
type Client struct{}

// Creates a SQL Client from the provided Config.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Query(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	panic("sqldb: not yet implemented")
}

func (c *Client) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	panic("sqldb: not yet implemented")
}
