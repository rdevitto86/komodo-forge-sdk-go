package cloudmonitoring

// GCP Cloud Monitoring client — pairs with cloudlogging as the equivalent of aws/cloudwatch (metrics side).
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import (
	"context"
	"time"
)

type MetricPoint struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

type API interface {
	PutMetric(ctx context.Context, point MetricPoint) error
	PutMetrics(ctx context.Context, points []MetricPoint) error
}

type Config struct {
	ProjectID       string
	CredentialsJSON string
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) PutMetric(_ context.Context, _ MetricPoint) error {
	panic("cloudmonitoring: not implemented")
}
func (c *Client) PutMetrics(_ context.Context, _ []MetricPoint) error {
	panic("cloudmonitoring: not implemented")
}
