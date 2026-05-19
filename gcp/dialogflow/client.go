package dialogflow

// GCP Dialogflow CX / Contact Center AI — equivalent to aws/connect.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.

import "context"

type DetectIntentInput struct {
	SessionID string
	AgentID   string
	Text      string
	Language  string
}

type DetectIntentOutput struct {
	ResponseText string
	Intent       string
	Confidence   float32
	Parameters   map[string]any
}

type API interface {
	DetectIntent(ctx context.Context, input DetectIntentInput) (*DetectIntentOutput, error)
}

type Config struct {
	ProjectID       string
	Location        string
	CredentialsJSON string
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) DetectIntent(_ context.Context, _ DetectIntentInput) (*DetectIntentOutput, error) {
	panic("dialogflow: not implemented")
}
