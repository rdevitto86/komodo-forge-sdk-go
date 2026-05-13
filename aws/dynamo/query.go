package dynamo

import (
	"context"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type QueryInput struct {
	TableName              string
	IndexName              *string
	KeyConditionExpression string
	FilterExpression       *string
	ExpressionValues       map[string]types.AttributeValue
	ExpressionNames        map[string]string
	Limit                  *int32
	ScanIndexForward       *bool
	ExclusiveStartKey      map[string]types.AttributeValue
}

type QueryOutput struct {
	Items            []map[string]types.AttributeValue
	LastEvaluatedKey map[string]types.AttributeValue
	Count            int32
}

type ScanInput struct {
	TableName         string
	IndexName         *string
	FilterExpression  *string
	ExpressionValues  map[string]types.AttributeValue
	ExpressionNames   map[string]string
	Limit             *int32
	ExclusiveStartKey map[string]types.AttributeValue
}

type ScanOutput struct {
	Items            []map[string]types.AttributeValue
	LastEvaluatedKey map[string]types.AttributeValue
	Count            int32
}

func (c *Client) Query(ctx context.Context, input QueryInput) (*QueryOutput, error) {
	q := &dynamodb.QueryInput{
		TableName:              aws.String(input.TableName),
		KeyConditionExpression: aws.String(input.KeyConditionExpression),
	}
	if input.IndexName != nil {
		q.IndexName = input.IndexName
	}
	if input.FilterExpression != nil {
		q.FilterExpression = input.FilterExpression
	}
	if input.ExpressionValues != nil {
		q.ExpressionAttributeValues = input.ExpressionValues
	}
	if input.ExpressionNames != nil {
		q.ExpressionAttributeNames = input.ExpressionNames
	}
	if input.Limit != nil {
		q.Limit = input.Limit
	}
	if input.ScanIndexForward != nil {
		q.ScanIndexForward = input.ScanIndexForward
	}
	if input.ExclusiveStartKey != nil {
		q.ExclusiveStartKey = input.ExclusiveStartKey
	}

	result, err := c.db.Query(ctx, q)
	if err != nil {
		logger.Error("dynamodb failed to query", err)
		return nil, WrapError(err, "Query")
	}
	return &QueryOutput{
		Items:            result.Items,
		LastEvaluatedKey: result.LastEvaluatedKey,
		Count:            result.Count,
	}, nil
}

func (c *Client) QueryAs(ctx context.Context, input QueryInput, out any) (*QueryOutput, error) {
	result, err := c.Query(ctx, input)
	if err != nil {
		logger.Error("dynamodb failed to query", err)
		return nil, WrapError(err, "QueryAs")
	}
	if err := attributevalue.UnmarshalListOfMaps(result.Items, out); err != nil {
		logger.Error("dynamodb failed to unmarshal items", err)
		return nil, WrapError(err, "QueryAs unmarshal")
	}
	return result, nil
}

func (c *Client) QueryAll(ctx context.Context, input QueryInput) ([]map[string]types.AttributeValue, error) {
	var allItems []map[string]types.AttributeValue
	var lastKey map[string]types.AttributeValue

	for {
		input.ExclusiveStartKey = lastKey
		result, err := c.Query(ctx, input)
		if err != nil {
			logger.Error("dynamodb failed to query", err)
			return nil, WrapError(err, "QueryAll")
		}
		allItems = append(allItems, result.Items...)
		if result.LastEvaluatedKey == nil {
			break
		}
		lastKey = result.LastEvaluatedKey
	}
	return allItems, nil
}

func (c *Client) QueryAllAs(ctx context.Context, input QueryInput, out any) error {
	items, err := c.QueryAll(ctx, input)
	if err != nil {
		logger.Error("dynamodb failed to query", err)
		return WrapError(err, "QueryAllAs")
	}
	if err = attributevalue.UnmarshalListOfMaps(items, out); err != nil {
		logger.Error("dynamodb failed to unmarshal items", err)
		return WrapError(err, "QueryAllAs unmarshal")
	}
	return nil
}

func (c *Client) Scan(ctx context.Context, input ScanInput) (*ScanOutput, error) {
	s := &dynamodb.ScanInput{
		TableName: aws.String(input.TableName),
	}
	if input.IndexName != nil {
		s.IndexName = input.IndexName
	}
	if input.FilterExpression != nil {
		s.FilterExpression = input.FilterExpression
	}
	if input.ExpressionValues != nil {
		s.ExpressionAttributeValues = input.ExpressionValues
	}
	if input.ExpressionNames != nil {
		s.ExpressionAttributeNames = input.ExpressionNames
	}
	if input.Limit != nil {
		s.Limit = input.Limit
	}
	if input.ExclusiveStartKey != nil {
		s.ExclusiveStartKey = input.ExclusiveStartKey
	}

	result, err := c.db.Scan(ctx, s)
	if err != nil {
		logger.Error("dynamodb failed to scan", err)
		return nil, WrapError(err, "Scan")
	}
	return &ScanOutput{
		Items:            result.Items,
		LastEvaluatedKey: result.LastEvaluatedKey,
		Count:            result.Count,
	}, nil
}

func (c *Client) ScanAs(ctx context.Context, input ScanInput, out any) (*ScanOutput, error) {
	result, err := c.Scan(ctx, input)
	if err != nil {
		logger.Error("dynamodb failed to scan", err)
		return nil, WrapError(err, "ScanAs")
	}
	if err = attributevalue.UnmarshalListOfMaps(result.Items, out); err != nil {
		logger.Error("dynamodb failed to unmarshal items", err)
		return nil, WrapError(err, "ScanAs unmarshal")
	}
	return result, nil
}

func (c *Client) ScanAll(ctx context.Context, input ScanInput) ([]map[string]types.AttributeValue, error) {
	var allItems []map[string]types.AttributeValue
	var lastKey map[string]types.AttributeValue

	for {
		input.ExclusiveStartKey = lastKey
		result, err := c.Scan(ctx, input)
		if err != nil {
			logger.Error("dynamodb failed to scan", err)
			return nil, WrapError(err, "ScanAll")
		}
		allItems = append(allItems, result.Items...)
		if result.LastEvaluatedKey == nil {
			break
		}
		lastKey = result.LastEvaluatedKey
	}
	return allItems, nil
}

func (c *Client) ScanAllAs(ctx context.Context, input ScanInput, out any) error {
	items, err := c.ScanAll(ctx, input)
	if err != nil {
		logger.Error("dynamodb failed to scan all", err)
		return WrapError(err, "ScanAllAs")
	}
	if err = attributevalue.UnmarshalListOfMaps(items, out); err != nil {
		logger.Error("dynamodb failed to unmarshal items", err)
		return WrapError(err, "ScanAllAs unmarshal")
	}
	return nil
}
