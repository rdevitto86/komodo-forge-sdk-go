package bannedcustomers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamodb"
)

// embeds dynamodb.API so only BuildKey and GetItemAs need implementing
type fakeDynamoDB struct {
	dynamodb.API
	rec *record
	err error
}

func (f fakeDynamoDB) BuildKey(pk string, pv any, sk string, sv any) (map[string]types.AttributeValue, error) {
	return map[string]types.AttributeValue{pk: &types.AttributeValueMemberS{Value: pv.(string)}}, nil
}

func (f fakeDynamoDB) GetItemAs(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue, out any) error {
	if f.err != nil {
		return f.err
	}
	if f.rec == nil {
		return dynamodb.ErrNotFound
	}
	item, err := attributevalue.MarshalMap(f.rec)
	if err != nil {
		return err
	}
	return attributevalue.UnmarshalMap(item, out)
}

func newClient(t *testing.T, db dynamodb.API) *Client {
	t.Helper()
	c, err := New(Config{TableName: "banned-customers", DynamoDB: db})
	if err != nil {
		t.Fatalf("unexpected error constructing client: %v", err)
	}
	return c
}

func TestIsBanned_NotBanned(t *testing.T) {
	c := newClient(t, fakeDynamoDB{})

	banned, err := c.IsBanned(context.Background(), "nobody@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if banned {
		t.Error("expected unknown email to be reported as not banned")
	}
}

func TestIsBanned_BannedAndActive(t *testing.T) {
	rec := &record{Email: "banned@example.com", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	c := newClient(t, fakeDynamoDB{rec: rec})

	banned, err := c.IsBanned(context.Background(), "banned@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !banned {
		t.Error("expected active ban record to report banned")
	}
}

func TestIsBanned_BannedButExpired(t *testing.T) {
	rec := &record{Email: "expired@example.com", ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	c := newClient(t, fakeDynamoDB{rec: rec})

	banned, err := c.IsBanned(context.Background(), "expired@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if banned {
		t.Error("expected expired ban record to report not banned")
	}
}

func TestIsBanned_LookupErrorFailsOpen(t *testing.T) {
	c := newClient(t, fakeDynamoDB{err: errors.New("connection refused")})

	banned, err := c.IsBanned(context.Background(), "someone@example.com")
	if err != nil {
		t.Fatalf("expected lookup failure to fail open without an error, got %v", err)
	}
	if banned {
		t.Error("expected lookup failure to report not banned (fail open)")
	}
}

func TestIsBanned_EmptyEmailIsCallerError(t *testing.T) {
	c := newClient(t, fakeDynamoDB{})

	if _, err := c.IsBanned(context.Background(), ""); err == nil {
		t.Error("expected an empty email to return an error")
	}
}

func TestNew_RequiresTableNameAndClient(t *testing.T) {
	if _, err := New(Config{DynamoDB: fakeDynamoDB{}}); err == nil {
		t.Error("expected an error when TableName is missing")
	}
	if _, err := New(Config{TableName: "banned-customers"}); err == nil {
		t.Error("expected an error when DynamoDB is missing")
	}
}
