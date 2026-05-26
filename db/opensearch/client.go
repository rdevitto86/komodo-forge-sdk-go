package opensearch

import "context"

// Defines the OpenSearch operations provided by this package.
type API interface {
	Index(ctx context.Context, index, id string, doc any) error
	Search(ctx context.Context, index string, query any) ([]map[string]any, error)
	Delete(ctx context.Context, index, id string) error
}

// Holds connection parameters for an OpenSearch client.
type Config struct {
	Endpoint string
	Username string
	Password string
	Region   string
}

// Wraps an OpenSearch REST client.
type Client struct{}

// Creates an OpenSearch Client from the provided Config.
func New(config Config) (*Client, error) {
	return &Client{}, nil
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
