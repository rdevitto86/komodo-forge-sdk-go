package ccaiinsights

import "context"

type Conversation struct {
	Name       string
	Transcript string
	Metadata   map[string]string
}

type Analysis struct {
	Sentiment float32
	Topics    []string
	Entities  map[string]any
}

type API interface {
	AnalyzeConversation(ctx context.Context, conv Conversation) (*Analysis, error)
	GetAnalysis(ctx context.Context, name string) (*Analysis, error)
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

func (c *Client) AnalyzeConversation(_ context.Context, _ Conversation) (*Analysis, error) {
	panic("ccaiinsights: not implemented")
}
func (c *Client) GetAnalysis(_ context.Context, _ string) (*Analysis, error) {
	panic("ccaiinsights: not implemented")
}
