package secretmanager

// GCP Secret Manager client — equivalent to aws/secretsmanager.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import "context"

type API interface {
	GetSecret(ctx context.Context, name string) (string, error)
	GetSecrets(ctx context.Context, names []string) (map[string]string, error)
}

type Config struct {
	ProjectID       string
	CredentialsJSON string
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) GetSecret(_ context.Context, _ string) (string, error) {
	panic("secretmanager: not implemented")
}
func (c *Client) GetSecrets(_ context.Context, _ []string) (map[string]string, error) {
	panic("secretmanager: not implemented")
}
