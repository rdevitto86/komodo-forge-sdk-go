package cloudmonitoring

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
