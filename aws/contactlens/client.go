package contactlens

import "context"

type API interface {
	ListRealtimeContactAnalysisSegments(ctx context.Context, instanceID, contactID string) ([]Segment, error)
}

type Config struct {
	Region     string
	AccessKey  string
	SecretKey  string
	Endpoint   string
	InstanceID string
}

type Segment struct {
	Type    string
	Content string
}

// Client wraps the AWS Contact Lens SDK client.
// TODO: wire github.com/aws/aws-sdk-go-v2/service/contactlens when implementing.
type Client struct{}

// Creates and returns a new Contact Lens Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) ListRealtimeContactAnalysisSegments(ctx context.Context, instanceID, contactID string) ([]Segment, error) {
	panic("contactlens: not yet implemented")
}
