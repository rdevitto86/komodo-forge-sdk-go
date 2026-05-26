package pubsubpub

import "context"

type PublishInput struct {
	Topic       string
	Body        []byte
	Attributes  map[string]string
	OrderingKey string
}

type API interface {
	Publish(ctx context.Context, input PublishInput) (messageID string, err error)
}

type Config struct {
	ProjectID       string
	CredentialsJSON string
	Endpoint        string // for emulator
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) Publish(_ context.Context, _ PublishInput) (string, error) {
	panic("pubsubpub: not implemented")
}
