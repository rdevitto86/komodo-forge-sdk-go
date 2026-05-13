package cloudwatch

import "context"

type API interface {
	PutMetricData(ctx context.Context, namespace string, metrics []MetricDatum) error
	PutLogEvents(ctx context.Context, logGroup, logStream string, events []LogEvent) error
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type MetricDatum struct {
	Name       string
	Value      float64
	Unit       string
	Dimensions map[string]string
}

type LogEvent struct {
	Timestamp int64
	Message   string
}

// Client wraps the AWS CloudWatch SDK client.
// TODO: wire github.com/aws/aws-sdk-go-v2/service/cloudwatch when implementing.
type Client struct{}

// Creates and returns a new CloudWatch Client.
func New(config Config) (*Client, error) {
	return &Client{}, nil
}

func (c *Client) PutMetricData(ctx context.Context, namespace string, metrics []MetricDatum) error {
	panic("cloudwatch: not yet implemented")
}

func (c *Client) PutLogEvents(ctx context.Context, logGroup, logStream string, events []LogEvent) error {
	panic("cloudwatch: not yet implemented")
}
