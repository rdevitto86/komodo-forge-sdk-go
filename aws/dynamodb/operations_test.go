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

type fakeDynamoDBAPI struct {
	dynamoDBAPI
	batchWriteItemFunc func(ctx context.Context, input *dynamodb.BatchWriteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	batchGetItemFunc   func(ctx context.Context, input *dynamodb.BatchGetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
}

func (f fakeDynamoDBAPI) BatchWriteItem(ctx context.Context, input *dynamodb.BatchWriteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	return f.batchWriteItemFunc(ctx, input, opts...)
}

func (f fakeDynamoDBAPI) BatchGetItem(ctx context.Context, input *dynamodb.BatchGetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	return f.batchGetItemFunc(ctx, input, opts...)
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
				return &dynamodb.BatchWriteItemOutput{UnprocessedItems: input.RequestItems}, nil
			}
			return &dynamodb.BatchWriteItemOutput{}, nil
		},
	}
	c := newWithAPI(fake, 1)

	start := time.Now()
	err := c.retryUnprocessed(context.Background(), writeRequests(table, 1))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected retries to eventually succeed, got error: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("expected 3 BatchWriteItem calls (1 initial unprocessed report not counted here, 2 retries + 1 success), got %d", got)
	}
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

	err := c.retryUnprocessed(context.Background(), writeRequests(table, 1))
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

	err := c.retryUnprocessed(context.Background(), writeRequests("widgets", 1))
	if err == nil || !errors.Is(err, apiErr) {
		t.Fatalf("expected the API error to be wrapped and retrievable via errors.Is, got %v", err)
	}
}

func TestBatchGetItems_RetriesUnprocessedKeys(t *testing.T) {
	testutil.Component(t)

	const table = "widgets"
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "a"}}
	item := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "a"}, "v": &types.AttributeValueMemberN{Value: "1"}}

	var calls atomic.Int32
	fake := fakeDynamoDBAPI{
		batchGetItemFunc: func(_ context.Context, input *dynamodb.BatchGetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
			if calls.Add(1) == 1 {
				return &dynamodb.BatchGetItemOutput{
					Responses:       map[string][]map[string]types.AttributeValue{},
					UnprocessedKeys: input.RequestItems,
				}, nil
			}
			return &dynamodb.BatchGetItemOutput{
				Responses: map[string][]map[string]types.AttributeValue{table: {item}},
			}, nil
		},
	}
	c := newWithAPI(fake, 1)

	got, err := c.batchGetItems(context.Background(), table, []map[string]types.AttributeValue{key})
	if err != nil {
		t.Fatalf("expected batchGetItems to recover unprocessed keys, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 item after retrying unprocessed keys, got %d", len(got))
	}
	if calls.Load() != 2 {
		t.Errorf("expected 2 BatchGetItem calls (initial + 1 retry), got %d", calls.Load())
	}
}

func TestBatchGetItems_ExhaustsUnprocessedKeys(t *testing.T) {
	testutil.Component(t)

	const table = "widgets"
	key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "a"}}
	fake := fakeDynamoDBAPI{
		batchGetItemFunc: func(_ context.Context, input *dynamodb.BatchGetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
			return &dynamodb.BatchGetItemOutput{
				Responses:       map[string][]map[string]types.AttributeValue{},
				UnprocessedKeys: input.RequestItems,
			}, nil
		},
	}
	c := newWithAPI(fake, 1)

	if _, err := c.batchGetItems(context.Background(), table, []map[string]types.AttributeValue{key}); err == nil {
		t.Fatal("expected an error after exhausting unprocessed-key retries, got nil")
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
