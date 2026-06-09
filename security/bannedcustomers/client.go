package bannedcustomers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamodb"
	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"
)

type Checker interface {
	IsBanned(ctx context.Context, email string) (bool, error)
}

type Config struct {
	TableName string
	DynamoDB  dynamodb.API
	// FailOpen controls behaviour when a lookup errors: when nil or true (the default),
	// errors are swallowed so a DynamoDB outage never blocks legitimate customers; set it
	// to false for fraud/abuse controls that must fail closed (return the error so the
	// caller blocks the request).
	FailOpen *bool
}

// Looks up ban records in DynamoDB; fails open on lookup errors by default so an outage
// never blocks legitimate customers, unless Config.FailOpen is set to false.
type Client struct {
	table    string
	db       dynamodb.API
	failOpen bool
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
	failOpen := cfg.FailOpen == nil || *cfg.FailOpen
	return &Client{table: cfg.TableName, db: cfg.DynamoDB, failOpen: failOpen}, nil
}

// Treats expired ban records as inactive and fails open on lookup errors other than not-found.
func (c *Client) IsBanned(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, fmt.Errorf("requires a non-empty email")
	}

	key, err := c.db.BuildKey("email", email, "", nil)
	if err != nil {
		if c.failOpen {
			logger.Error("failed to build banned-customer lookup key; failing open", err)
			return false, nil
		}
		return false, fmt.Errorf("failed to build banned-customer lookup key: %w", err)
	}

	var rec record
	if err := c.db.GetItemAs(ctx, c.table, key, false, nil, &rec); err != nil {
		if errors.Is(err, dynamodb.ErrNotFound) {
			return false, nil
		}
		if c.failOpen {
			logger.Error("banned-customer lookup failed; failing open", err)
			return false, nil
		}
		return false, fmt.Errorf("banned-customer lookup failed: %w", err)
	}

	if rec.ExpiresAt > 0 && rec.ExpiresAt <= time.Now().Unix() {
		return false, nil
	}
	return true, nil
}
