package dynamodb

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rdevitto86/komodo-forge-sdk-go/testing/testutil"
)

// embeds dynamoDBAPI so only the methods under test need implementing
type fakeDynamoDBAPI struct {
	dynamoDBAPI
	batchWriteItemFunc func(ctx context.Context, input *dynamodb.BatchWriteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
}

func (f fakeDynamoDBAPI) BatchWriteItem(ctx context.Context, input *dynamodb.BatchWriteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	return f.batchWriteItemFunc(ctx, input, opts...)
}

func writeRequests(table string, n int) map[string][]types.WriteRequest {
	reqs := make([]types.WriteRequest, n)
	for i := range reqs {
		reqs[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "item"},
		}}}
	}
	return map[string][]types.WriteRequest{table: reqs}
}

func TestRetryUnprocessed_SucceedsAfterRetries(t *testing.T) {
	testutil.Component(t)

	const table = "widgets"
	var calls atomic.Int32
	fake := fakeDynamoDBAPI{
		batchWriteItemFunc: func(_ context.Context, input *dynamodb.BatchWriteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
			n := calls.Add(1)
			if n < 3 {
				// still throttled — return the same unprocessed batch
				return &dynamodb.BatchWriteItemOutput{UnprocessedItems: input.RequestItems}, nil
			}
			return &dynamodb.BatchWriteItemOutput{}, nil
		},
	}
	c := newWithAPI(fake, 1)

	start := time.Now()
	err := c.retryUnprocessed(context.Background(), "batchWriteItem", writeRequests(table, 1))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected retries to eventually succeed, got error: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("expected 3 BatchWriteItem calls (1 initial unprocessed report not counted here, 2 retries + 1 success), got %d", got)
	}
	// Two retries with base 50ms doubling: ~50ms + ~100ms = ~150ms minimum.
	if elapsed < 140*time.Millisecond {
		t.Errorf("expected exponential backoff between retries, elapsed only %v", elapsed)
	}
}

func TestRetryUnprocessed_ExhaustsAndWrapsError(t *testing.T) {
	testutil.Component(t)

	const table = "widgets"
	var calls atomic.Int32
	fake := fakeDynamoDBAPI{
		batchWriteItemFunc: func(_ context.Context, input *dynamodb.BatchWriteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
			calls.Add(1)
			return &dynamodb.BatchWriteItemOutput{UnprocessedItems: input.RequestItems}, nil
		},
	}
	c := newWithAPI(fake, 1)

	err := c.retryUnprocessed(context.Background(), "batchWriteItem", writeRequests(table, 1))
	if err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	if got := calls.Load(); got != maxBatchRetries {
		t.Errorf("expected exactly %d retry attempts, got %d", maxBatchRetries, got)
	}
}

func TestRetryUnprocessed_PropagatesAPIError(t *testing.T) {
	testutil.Component(t)

	apiErr := errors.New("throttled")
	fake := fakeDynamoDBAPI{
		batchWriteItemFunc: func(_ context.Context, _ *dynamodb.BatchWriteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
			return nil, apiErr
		},
	}
	c := newWithAPI(fake, 1)

	err := c.retryUnprocessed(context.Background(), "batchWriteItem", writeRequests("widgets", 1))
	if err == nil || !errors.Is(err, apiErr) {
		t.Fatalf("expected the API error to be wrapped and retrievable via errors.Is, got %v", err)
	}
}

func TestBatchWriteItems_RetriesUnprocessedThenSucceeds(t *testing.T) {
	testutil.Component(t)

	const table = "widgets"
	var calls atomic.Int32
	fake := fakeDynamoDBAPI{
		batchWriteItemFunc: func(_ context.Context, input *dynamodb.BatchWriteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
			if calls.Add(1) == 1 {
				return &dynamodb.BatchWriteItemOutput{UnprocessedItems: input.RequestItems}, nil
			}
			return &dynamodb.BatchWriteItemOutput{}, nil
		},
	}
	c := newWithAPI(fake, 1)

	items := []map[string]types.AttributeValue{{"id": &types.AttributeValueMemberS{Value: "item"}}}
	if err := c.batchWriteItems(context.Background(), table, items); err != nil {
		t.Fatalf("expected batchWriteItems to succeed after retrying unprocessed items, got: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 BatchWriteItem calls (initial + 1 retry), got %d", got)
	}
}
