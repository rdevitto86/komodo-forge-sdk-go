package dynamo

import (
	"context"
	"fmt"
	"sync"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func (c *Client) getItem(
	ctx context.Context,
	tableName string,
	key map[string]types.AttributeValue,
) (map[string]types.AttributeValue, error) {
	result, err := c.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})
	if err != nil {
		logger.Error("failed to get item", err)
		return nil, WrapError(err, "getItem")
	}
	if result.Item == nil {
		return nil, WrapError(fmt.Errorf("item not found"), "getItem")
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
		return WrapError(err, "putItem")
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
		return WrapError(err, "deleteItem")
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

// runParallel runs fn for each chunk index in parallel, bounded by the client's
// maxParallel semaphore. Returns the first error encountered.
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
			return WrapError(err, "batchGetItems")
		}
		if len(result.UnprocessedKeys) > 0 {
			return WrapError(fmt.Errorf("batch get has unprocessed keys"), "batchGetItems")
		}
		results[i] = result.Responses[tableName]
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
		return WrapError(err, "batchGetItemsAs")
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
			return WrapError(err, "batchWriteItem")
		}
		if len(result.UnprocessedItems) > 0 {
			return WrapError(fmt.Errorf("batch write has unprocessed items"), "batchWriteItem")
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
			return WrapError(err, "batchDeleteItem")
		}
		if len(result.UnprocessedItems) > 0 {
			return WrapError(fmt.Errorf("batch delete has unprocessed items"), "batchDeleteItem")
		}
		return nil
	})
}
