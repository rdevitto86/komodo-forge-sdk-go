package vertexsearch

// GCP Vertex AI Search client — equivalent to aws/elasticsearch.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import "context"

type SearchInput struct {
	DataStore string
	Query     string
	PageSize  int32
	Filter    string
}

type SearchResult struct {
	ID    string
	Score float32
	Doc   map[string]any
}

type API interface {
	Search(ctx context.Context, input SearchInput) ([]SearchResult, error)
	Index(ctx context.Context, dataStore, id string, doc map[string]any) error
	Delete(ctx context.Context, dataStore, id string) error
}

type Config struct {
	ProjectID       string
	Location        string
	CredentialsJSON string
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Search(_ context.Context, _ SearchInput) ([]SearchResult, error) {
	panic("vertexsearch: not implemented")
}
func (c *Client) Index(_ context.Context, _, _ string, _ map[string]any) error {
	panic("vertexsearch: not implemented")
}
func (c *Client) Delete(_ context.Context, _, _ string) error {
	panic("vertexsearch: not implemented")
}
