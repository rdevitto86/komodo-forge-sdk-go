package cloudsql

import (
	"context"
	"database/sql"
)

type API interface {
	DB() *sql.DB
	Ping(ctx context.Context) error
	Close() error
}

type Config struct {
	InstanceConnectionName string // project:region:instance
	Database               string
	User                   string
	Password               string
	IAMAuth                bool
	MaxOpenConns           int
	MaxIdleConns           int
	CredentialsJSON        string
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) DB() *sql.DB                  { panic("cloudsql: not implemented") }
func (c *Client) Ping(_ context.Context) error { panic("cloudsql: not implemented") }
func (c *Client) Close() error                 { panic("cloudsql: not implemented") }
