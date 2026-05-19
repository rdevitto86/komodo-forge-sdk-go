package firestore

// GCP Firestore client — equivalent to aws/dynamodb.
//
// Stub: methods panic until implementation lands. New returns ErrNotImplemented.
// Firestore is document-oriented; the API surface intentionally mirrors the
// DynamoDB client so callers can swap providers by import path. Some methods
// (Scan, conditional expressions) map awkwardly and are documented in TODO.md.

import "context"

type Item map[string]any

type API interface {
	GetItem(ctx context.Context, collection, id string) (Item, error)
	GetItemAs(ctx context.Context, collection, id string, out any) error
	PutItem(ctx context.Context, collection, id string, item Item) error
	PutItemFrom(ctx context.Context, collection, id string, item any) error
	UpdateItem(ctx context.Context, collection, id string, updates Item) (Item, error)
	DeleteItem(ctx context.Context, collection, id string) error
	Query(ctx context.Context, collection string, where map[string]any, limit int) ([]Item, error)
	QueryAs(ctx context.Context, collection string, where map[string]any, limit int, out any) error
}

type Config struct {
	ProjectID       string
	DatabaseID      string // defaults to "(default)"
	CredentialsJSON string
	Endpoint        string // for emulator
}

type Client struct{}

func New(_ Config) (*Client, error) {
	return nil, ErrNotImplemented
}

func (c *Client) GetItem(_ context.Context, _, _ string) (Item, error) {
	panic("firestore: not implemented")
}
func (c *Client) GetItemAs(_ context.Context, _, _ string, _ any) error {
	panic("firestore: not implemented")
}
func (c *Client) PutItem(_ context.Context, _, _ string, _ Item) error {
	panic("firestore: not implemented")
}
func (c *Client) PutItemFrom(_ context.Context, _, _ string, _ any) error {
	panic("firestore: not implemented")
}
func (c *Client) UpdateItem(_ context.Context, _, _ string, _ Item) (Item, error) {
	panic("firestore: not implemented")
}
func (c *Client) DeleteItem(_ context.Context, _, _ string) error {
	panic("firestore: not implemented")
}
func (c *Client) Query(_ context.Context, _ string, _ map[string]any, _ int) ([]Item, error) {
	panic("firestore: not implemented")
}
func (c *Client) QueryAs(_ context.Context, _ string, _ map[string]any, _ int, _ any) error {
	panic("firestore: not implemented")
}
