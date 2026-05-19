package gcs

// GCP Cloud Storage client — equivalent to aws/s3.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import (
	"context"
)

type API interface {
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
	GetObjectAs(ctx context.Context, bucket, key string, out any) error
	PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error
	DeleteObject(ctx context.Context, bucket, key string) error
}

type Config struct {
	ProjectID       string
	CredentialsJSON string // path or inline JSON; empty = ADC
	Endpoint        string // for emulator
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) GetObject(_ context.Context, _, _ string) ([]byte, error) {
	panic("gcs: not implemented")
}

func (c *Client) GetObjectAs(_ context.Context, _, _ string, _ any) error {
	panic("gcs: not implemented")
}

func (c *Client) PutObject(_ context.Context, _, _ string, _ []byte, _ string) error {
	panic("gcs: not implemented")
}

func (c *Client) DeleteObject(_ context.Context, _, _ string) error {
	panic("gcs: not implemented")
}
