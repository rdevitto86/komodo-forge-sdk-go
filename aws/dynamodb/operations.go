package dynamodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (c *Client) DescribeTable(ctx context.Context, table string) error {
	if _, err := c.db.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)}); err != nil {
		logger.Error("failed to describe table", err)
		return WrapError(err)
	}
	return nil
}

func (c *Client) getItem(ctx context.Context, tableName string, key map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	result, err := c.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})
	if err != nil {
		logger.Error("failed to get item", err)
		return nil, WrapError(err)
	}
	if result.Item == nil {
		return nil, ErrNotFound
	}
	return result.Item, nil
}

func (c *Client) putItem(
	ctx context.Context,
	tableName string,
	item map[string]types.AttributeValue,
	condition *string,
) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	}
	if condition != nil {
		input.ConditionExpression = condition
	}
	if _, err := c.db.PutItem(ctx, input); err != nil {
		logger.Error("failed to put item", err)
		return WrapError(err)
	}
	return nil
}

func (c *Client) deleteItem(
	ctx context.Context,
	tableName string,
	key map[string]types.AttributeValue,
	condition *string,
) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	}
	if condition != nil {
		input.ConditionExpression = condition
	}
	if _, err := c.db.DeleteItem(ctx, input); err != nil {
		logger.Error("failed to delete item", err)
		return WrapError(err)
	}
	return nil
}

func chunks[T any](items []T) [][]T {
	var out [][]T
	for i := 0; i < len(items); i += maxBatchSize {
		end := min(i+maxBatchSize, len(items))
		out = append(out, items[i:end])
	}
	return out
}

func (c *Client) runParallel(n int, fn func(i int) error) error {
	if n == 1 {
		return fn(0)
	}

	sem := make(chan struct{}, c.maxParallel)
	errs := make([]error, n)
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			errs[idx] = fn(idx)
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	maxBatchRetries     = 5
	batchRetryBaseDelay = 50 * time.Millisecond
)

func (c *Client) retryUnprocessed(ctx context.Context, unprocessed map[string][]types.WriteRequest) error {
	delay := batchRetryBaseDelay
	for attempt := 1; len(unprocessed) > 0; attempt++ {
		if attempt > maxBatchRetries {
			return WrapError(fmt.Errorf("unprocessed write items remain after %d retries", maxBatchRetries))
		}

		select {
		case <-ctx.Done():
			return WrapError(ctx.Err())
		case <-time.After(delay):
		}
		delay *= 2

		result, err := c.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{RequestItems: unprocessed})
		if err != nil {
			logger.Error("retry of unprocessed batch items failed", err)
			return WrapError(err)
		}
		unprocessed = result.UnprocessedItems
	}
	return nil
}

func (c *Client) retryUnprocessedKeys(ctx context.Context, tableName string, unprocessed map[string]types.KeysAndAttributes) ([]map[string]types.AttributeValue, error) {
	var items []map[string]types.AttributeValue
	delay := batchRetryBaseDelay
	for attempt := 1; len(unprocessed) > 0; attempt++ {
		if attempt > maxBatchRetries {
			return nil, WrapError(fmt.Errorf("unprocessed read keys remain after %d retries", maxBatchRetries))
		}

		select {
		case <-ctx.Done():
			return nil, WrapError(ctx.Err())
		case <-time.After(delay):
		}
		delay *= 2

		result, err := c.db.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{RequestItems: unprocessed})
		if err != nil {
			logger.Error("retry of unprocessed batch keys failed", err)
			return nil, WrapError(err)
		}
		items = append(items, result.Responses[tableName]...)
		unprocessed = result.UnprocessedKeys
	}
	return items, nil
}

func (c *Client) batchGetItems(
	ctx context.Context,
	tableName string,
	keys []map[string]types.AttributeValue,
) ([]map[string]types.AttributeValue, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	batches := chunks(keys)
	results := make([][]map[string]types.AttributeValue, len(batches))

	err := c.runParallel(len(batches), func(i int) error {
		result, err := c.db.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
			RequestItems: map[string]types.KeysAndAttributes{
				tableName: {Keys: batches[i]},
			},
		})
		if err != nil {
			logger.Error("failed to batch get items", err)
			return WrapError(err)
		}
		items := result.Responses[tableName]
		if len(result.UnprocessedKeys) > 0 {
			retried, rerr := c.retryUnprocessedKeys(ctx, tableName, result.UnprocessedKeys)
			if rerr != nil {
				return rerr
			}
			items = append(items, retried...)
		}
		results[i] = items
		return nil
	})
	if err != nil {
		return nil, err
	}

	var allItems []map[string]types.AttributeValue
	for _, r := range results {
		allItems = append(allItems, r...)
	}
	return allItems, nil
}

func (c *Client) batchGetItemsAs(
	ctx context.Context,
	tableName string,
	keys []map[string]types.AttributeValue,
	out any,
) error {
	items, err := c.batchGetItems(ctx, tableName, keys)
	if err != nil {
		return err
	}
	if err = attributevalue.UnmarshalListOfMaps(items, out); err != nil {
		logger.Error("failed to unmarshal items", err)
		return WrapError(err)
	}
	return nil
}

func (c *Client) batchWriteItems(
	ctx context.Context,
	tableName string,
	items []map[string]types.AttributeValue,
) error {
	if len(items) == 0 {
		return nil
	}

	batches := chunks(items)

	return c.runParallel(len(batches), func(i int) error {
		writeRequests := make([]types.WriteRequest, len(batches[i]))
		for j, item := range batches[i] {
			writeRequests[j] = types.WriteRequest{
				PutRequest: &types.PutRequest{Item: item},
			}
		}

		result, err := c.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{tableName: writeRequests},
		})
		if err != nil {
			logger.Error("failed to batch write items", err)
			return WrapError(err)
		}
		if len(result.UnprocessedItems) > 0 {
			return c.retryUnprocessed(ctx, result.UnprocessedItems)
		}
		return nil
	})
}

func (c *Client) batchDeleteItems(
	ctx context.Context,
	tableName string,
	keys []map[string]types.AttributeValue,
) error {
	if len(keys) == 0 {
		return nil
	}

	batches := chunks(keys)

	return c.runParallel(len(batches), func(i int) error {
		writeRequests := make([]types.WriteRequest, len(batches[i]))
		for j, key := range batches[i] {
			writeRequests[j] = types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{Key: key},
			}
		}

		result, err := c.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{tableName: writeRequests},
		})
		if err != nil {
			logger.Error("failed to batch delete items", err)
			return WrapError(err)
		}
		if len(result.UnprocessedItems) > 0 {
			return c.retryUnprocessed(ctx, result.UnprocessedItems)
		}
		return nil
	})
}
