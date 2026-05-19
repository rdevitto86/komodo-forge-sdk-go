package vertexai

// GCP Vertex AI client — equivalent to aws/bedrock.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import "context"

type InvokeInput struct {
	Model       string
	Prompt      string
	Parameters  map[string]any
	System      string
}

type InvokeOutput struct {
	Text     string
	Raw      []byte
	Usage    map[string]int
}

type API interface {
	Invoke(ctx context.Context, input InvokeInput) (*InvokeOutput, error)
	InvokeStream(ctx context.Context, input InvokeInput, onChunk func([]byte) error) error
}

type Config struct {
	ProjectID       string
	Region          string
	CredentialsJSON string
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Invoke(_ context.Context, _ InvokeInput) (*InvokeOutput, error) {
	panic("vertexai: not implemented")
}
func (c *Client) InvokeStream(_ context.Context, _ InvokeInput, _ func([]byte) error) error {
	panic("vertexai: not implemented")
}
