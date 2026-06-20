package opensearch

import "context"

type API interface {
	Index(ctx context.Context, index, id string, doc any) error
	Search(ctx context.Context, index string, query any) ([]map[string]any, error)
	Delete(ctx context.Context, index, id string) error
}

type Config struct {
	Endpoint string
	Username string
	Password string
	Region   string
}

type Client struct{}

func New(config Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Index(ctx context.Context, index, id string, doc any) error {
	panic("opensearch: not yet implemented")
}

func (c *Client) Search(ctx context.Context, index string, query any) ([]map[string]any, error) {
	panic("opensearch: not yet implemented")
}

func (c *Client) Delete(ctx context.Context, index, id string) error {
	panic("opensearch: not yet implemented")
}
