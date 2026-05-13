package bedrock

import "context"

// API is the interface for AWS Bedrock operations.
type API interface {
	InvokeModel(ctx context.Context, modelID string, body []byte) ([]byte, error)
	InvokeModelStream(ctx context.Context, modelID string, body []byte) (<-chan []byte, error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

// Client wraps the AWS Bedrock Runtime SDK client.
// TODO: wire github.com/aws/aws-sdk-go-v2/service/bedrockruntime when implementing.
type Client struct{}

// New creates and returns a new Bedrock Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) InvokeModel(ctx context.Context, modelID string, body []byte) ([]byte, error) {
	panic("bedrock: not yet implemented")
}

func (c *Client) InvokeModelStream(ctx context.Context, modelID string, body []byte) (<-chan []byte, error) {
	panic("bedrock: not yet implemented")
}
