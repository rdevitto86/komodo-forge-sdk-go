package cloudlogging

import "context"

type Entry struct {
	LogName  string
	Severity string
	Payload  any
	Labels   map[string]string
}

type API interface {
	Write(ctx context.Context, entry Entry) error
	WriteBatch(ctx context.Context, entries []Entry) error
}

type Config struct {
	ProjectID       string
	CredentialsJSON string
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Write(_ context.Context, _ Entry) error {
	panic("cloudlogging: not implemented")
}
func (c *Client) WriteBatch(_ context.Context, _ []Entry) error {
	panic("cloudlogging: not implemented")
}
