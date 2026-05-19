package cloudfunctions

// GCP Cloud Functions / Cloud Run client — equivalent to aws/lambda.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import "context"

type InvokeInput struct {
	FunctionName string
	Payload      []byte
	Async        bool
}

type API interface {
	Invoke(ctx context.Context, input InvokeInput) ([]byte, error)
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

func (c *Client) Invoke(_ context.Context, _ InvokeInput) ([]byte, error) {
	panic("cloudfunctions: not implemented")
}
