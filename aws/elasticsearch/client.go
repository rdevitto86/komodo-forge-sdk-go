package elasticsearch

import "context"

// API is the interface for Elasticsearch/OpenSearch operations.
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

// Client wraps an Elasticsearch/OpenSearch REST client.
// TODO: wire opensearch-go or olivere/elastic when implementing.
type Client struct{}

// New creates and returns a new Elasticsearch Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Index(ctx context.Context, index, id string, doc any) error {
	panic("elasticsearch: not yet implemented")
}

func (c *Client) Search(ctx context.Context, index string, query any) ([]map[string]any, error) {
	panic("elasticsearch: not yet implemented")
}

func (c *Client) Delete(ctx context.Context, index, id string) error {
	panic("elasticsearch: not yet implemented")
}
