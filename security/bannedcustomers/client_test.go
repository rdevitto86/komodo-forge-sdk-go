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

func TestIsBanned_FailClosedReturnsError(t *testing.T) {
	failClosed := false
	c, err := New(Config{
		TableName: "banned-customers",
		DynamoDB:  fakeDynamoDB{err: errors.New("connection refused")},
		FailOpen:  &failClosed,
	})
	if err != nil {
		t.Fatalf("unexpected error constructing client: %v", err)
	}

	if _, err := c.IsBanned(context.Background(), "someone@example.com"); err == nil {
		t.Error("expected fail-closed lookup failure to return an error")
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

type capturingDynamoDB struct {
	dynamodb.API
	gotEmail string
}

func (f *capturingDynamoDB) BuildKey(pk string, pv any, sk string, sv any) (map[string]types.AttributeValue, error) {
	f.gotEmail = pv.(string)
	return map[string]types.AttributeValue{pk: &types.AttributeValueMemberS{Value: pv.(string)}}, nil
}

func (f *capturingDynamoDB) GetItemAs(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue, out any) error {
	return dynamodb.ErrNotFound
}

func TestIsBanned_NormalizesEmail(t *testing.T) {
	fake := &capturingDynamoDB{}
	c, err := New(Config{TableName: "banned-customers", DynamoDB: fake})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.IsBanned(context.Background(), "  User@Example.COM  "); err != nil {
		t.Fatalf("IsBanned: %v", err)
	}
	if fake.gotEmail != "user@example.com" {
		t.Errorf("lookup email = %q, want normalized user@example.com", fake.gotEmail)
	}
}

type countingDynamoDB struct {
	dynamodb.API
	rec   *record
	calls int
}

func (f *countingDynamoDB) BuildKey(pk string, pv any, sk string, sv any) (map[string]types.AttributeValue, error) {
	return map[string]types.AttributeValue{pk: &types.AttributeValueMemberS{Value: pv.(string)}}, nil
}

func (f *countingDynamoDB) GetItemAs(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue, out any) error {
	f.calls++
	if f.rec == nil {
		return dynamodb.ErrNotFound
	}
	item, err := attributevalue.MarshalMap(f.rec)
	if err != nil {
		return err
	}
	return attributevalue.UnmarshalMap(item, out)
}

func TestIsBanned_CacheServesRepeatLookups(t *testing.T) {
	fake := &countingDynamoDB{rec: &record{ExpiresAt: time.Now().Add(time.Hour).Unix()}}
	c, err := New(Config{TableName: "t", DynamoDB: fake, CacheTTL: time.Minute})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := 0; i < 3; i++ {
		banned, err := c.IsBanned(context.Background(), "a@b.com")
		if err != nil || !banned {
			t.Fatalf("IsBanned = %v, %v; want true, nil", banned, err)
		}
	}
	if fake.calls != 1 {
		t.Errorf("expected 1 db call with caching, got %d", fake.calls)
	}
}

func TestIsBanned_NoCacheQueriesEachTime(t *testing.T) {
	fake := &countingDynamoDB{rec: &record{ExpiresAt: time.Now().Add(time.Hour).Unix()}}
	c, err := New(Config{TableName: "t", DynamoDB: fake})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := c.IsBanned(context.Background(), "a@b.com"); err != nil {
			t.Fatalf("IsBanned: %v", err)
		}
	}
	if fake.calls != 3 {
		t.Errorf("expected 3 db calls without caching, got %d", fake.calls)
	}
}
