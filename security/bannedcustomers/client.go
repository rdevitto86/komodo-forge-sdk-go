package bannedcustomers

import (
	"context"
	"fmt"
	"time"

	"errors"

	"github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamodb"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

type Checker interface {
	IsBanned(ctx context.Context, email string) (bool, error)
}

type Config struct {
	TableName string
	DynamoDB  dynamodb.API
}

// fails open on lookup errors so a DynamoDB outage never blocks legitimate customers
type Client struct {
	table string
	db    dynamodb.API
}

type record struct {
	Email     string `dynamodbav:"email"`
	ExpiresAt int64  `dynamodbav:"expires_at"`
}

// Creates a Client; both TableName and DynamoDB are required.
func New(cfg Config) (*Client, error) {
	if cfg.TableName == "" {
		return nil, errors.New("missing table name")
	}
	if cfg.DynamoDB == nil {
		return nil, errors.New("missing dynamodb client")
	}
	return &Client{table: cfg.TableName, db: cfg.DynamoDB}, nil
}

// Treats expired ban records as inactive and fails open on lookup errors other than not-found.
func (c *Client) IsBanned(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, fmt.Errorf("requires a non-empty email")
	}

	key, err := c.db.BuildKey("email", email, "", nil)
	if err != nil {
		logger.Error("failed to build banned-customer lookup key; failing open", err)
		return false, nil
	}

	var rec record
	if err := c.db.GetItemAs(ctx, c.table, key, false, nil, &rec); err != nil {
		if errors.Is(err, dynamodb.ErrNotFound) {
			return false, nil
		}
		logger.Error("banned-customer lookup failed; failing open", err)
		return false, nil
	}

	if rec.ExpiresAt > 0 && rec.ExpiresAt <= time.Now().Unix() {
		return false, nil
	}
	return true, nil
}
