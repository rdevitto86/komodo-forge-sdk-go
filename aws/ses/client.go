package ses

import "context"

// API is the interface for SES operations.
type API interface {
	SendEmail(ctx context.Context, from, to, subject, htmlBody, textBody string) (string, error)
	SendTemplatedEmail(ctx context.Context, from, to, templateName string, data map[string]any) (string, error)
}

type Config struct {
	Region      string
	AccessKey   string
	SecretKey   string
	Endpoint    string
	FromAddress string
}

// Client wraps the AWS SES SDK client.
// TODO: wire github.com/aws/aws-sdk-go-v2/service/ses (or sesv2) when implementing.
type Client struct{}

// New creates and returns a new SES Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) SendEmail(ctx context.Context, from, to, subject, htmlBody, textBody string) (string, error) {
	panic("ses: not yet implemented")
}

func (c *Client) SendTemplatedEmail(ctx context.Context, from, to, templateName string, data map[string]any) (string, error) {
	panic("ses: not yet implemented")
}
