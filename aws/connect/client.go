package connect

import "context"

type API interface {
	StartContactRecording(ctx context.Context, instanceID, contactID, participantID string) error
	StopContactRecording(ctx context.Context, instanceID, contactID, participantID string) error
}

type Config struct {
	Region     string
	AccessKey  string
	SecretKey  string
	Endpoint   string
	InstanceID string
}

// Client wraps the AWS Connect SDK client.
// TODO: wire github.com/aws/aws-sdk-go-v2/service/connect when implementing.
type Client struct{}

// Creates and returns a new Connect Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) StartContactRecording(ctx context.Context, instanceID, contactID, participantID string) error {
	panic("connect: not yet implemented")
}

func (c *Client) StopContactRecording(ctx context.Context, instanceID, contactID, participantID string) error {
	panic("connect: not yet implemented")
}
