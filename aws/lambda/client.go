package lambda

import "context"

type API interface {
	Invoke(ctx context.Context, functionName string, payload []byte) ([]byte, error)
	InvokeAsync(ctx context.Context, functionName string, payload []byte) error
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

// Client wraps the AWS Lambda SDK client.
// TODO: wire github.com/aws/aws-sdk-go-v2/service/lambda when implementing.
type Client struct{}

// Creates and returns a new Lambda Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Invoke(ctx context.Context, functionName string, payload []byte) ([]byte, error) {
	panic("lambda: not yet implemented")
}

func (c *Client) InvokeAsync(ctx context.Context, functionName string, payload []byte) error {
	panic("lambda: not yet implemented")
}
