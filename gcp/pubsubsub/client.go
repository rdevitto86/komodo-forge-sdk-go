package pubsubsub

// GCP Pub/Sub subscriber — equivalent to aws/sqs (pull-mode subscription).
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import "context"

type Message struct {
	ID          string
	Body        []byte
	AckID       string
	Attributes  map[string]string
	OrderingKey string
}

type API interface {
	Receive(ctx context.Context, subscription string, maxMessages int32) ([]Message, error)
	Ack(ctx context.Context, subscription, ackID string) error
	Nack(ctx context.Context, subscription, ackID string) error
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

func (c *Client) Receive(_ context.Context, _ string, _ int32) ([]Message, error) {
	panic("pubsubsub: not implemented")
}
func (c *Client) Ack(_ context.Context, _, _ string) error {
	panic("pubsubsub: not implemented")
}
func (c *Client) Nack(_ context.Context, _, _ string) error {
	panic("pubsubsub: not implemented")
}
