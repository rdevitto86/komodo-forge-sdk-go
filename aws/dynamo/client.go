package dynamo

import (
	"context"
	"fmt"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// API is the interface for all DynamoDB operations. Use it in callers so the
// concrete *Client can be swapped for a test double without modifying call sites.
type API interface {
	BuildKey(pk string, pv any, sk string, sv any) (map[string]types.AttributeValue, error)
	GetItem(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue) (any, error)
	GetItemAs(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue, out any) error
	WriteItem(ctx context.Context, table string, item map[string]types.AttributeValue, batch bool, items []map[string]types.AttributeValue, cond *string) error
	WriteItemFrom(ctx context.Context, table string, item any, batch bool, items any, cond *string) error
	UpdateItem(ctx context.Context, table string, key map[string]types.AttributeValue, expr string, exprVals map[string]types.AttributeValue, exprNames map[string]string, cond *string) (map[string]types.AttributeValue, error)
	UpdateItemAs(ctx context.Context, table string, key map[string]types.AttributeValue, expr string, exprVals map[string]types.AttributeValue, exprNames map[string]string, cond *string, out any) error
	DeleteItem(ctx context.Context, table string, key map[string]types.AttributeValue, batch bool, keys []map[string]types.AttributeValue, cond *string) error
	Query(ctx context.Context, input QueryInput) (*QueryOutput, error)
	QueryAs(ctx context.Context, input QueryInput, out any) (*QueryOutput, error)
	QueryAll(ctx context.Context, input QueryInput) ([]map[string]types.AttributeValue, error)
	QueryAllAs(ctx context.Context, input QueryInput, out any) error
	Scan(ctx context.Context, input ScanInput) (*ScanOutput, error)
	ScanAs(ctx context.Context, input ScanInput, out any) (*ScanOutput, error)
	ScanAll(ctx context.Context, input ScanInput) ([]map[string]types.AttributeValue, error)
	ScanAllAs(ctx context.Context, input ScanInput, out any) error
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
	// MaxConcurrentBatches caps the number of parallel DynamoDB batch requests
	// sent when an input is split into multiple 25-item chunks. Defaults to 5
	// when 0. Set to 1 to restore serial behaviour.
	MaxConcurrentBatches int
}

// Client wraps the AWS DynamoDB SDK client.
type Client struct {
	db          *dynamodb.Client
	maxParallel int // semaphore cap for batch operations
}

// New creates and returns a new DynamoDB Client. Returns an error if the AWS
// config cannot be loaded or the region is missing.
func New(config Config) (*Client, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("dynamodb: region is required")
	}

	ctx := context.Background()
	var (
		cfg aws.Config
		err error
	)

	if config.AccessKey != "" && config.SecretKey != "" {
		cfg, err = awsconfig.LoadDefaultConfig(
			ctx,
			awsconfig.WithRegion(config.Region),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, ""),
			),
		)
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	}

	if err != nil {
		logger.Error("dynamodb failed to load config", err)
		return nil, WrapError(err, "New")
	}

	var opts []func(*dynamodb.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	maxParallel := config.MaxConcurrentBatches
	if maxParallel <= 0 {
		maxParallel = 5
	}
	return &Client{db: dynamodb.NewFromConfig(cfg, opts...), maxParallel: maxParallel}, nil
}

const maxBatchSize = 25

func (c *Client) BuildKey(
	partitionKey string,
	partitionValue any,
	sortKey string,
	sortValue any,
) (map[string]types.AttributeValue, error) {
	key := make(map[string]types.AttributeValue)

	pkAttr, err := attributevalue.Marshal(partitionValue)
	if err != nil {
		logger.Error("failed to marshal partition key", err)
		return nil, WrapError(err, "BuildKey marshal partition key")
	}
	key[partitionKey] = pkAttr

	if sortKey != "" && sortValue != nil {
		skAttr, err := attributevalue.Marshal(sortValue)
		if err != nil {
			logger.Error("failed to marshal sort key", err)
			return nil, WrapError(err, "BuildKey marshal sort key")
		}
		key[sortKey] = skAttr
	}
	return key, nil
}

func (c *Client) GetItem(
	ctx context.Context,
	tableName string,
	key map[string]types.AttributeValue,
	batch bool,
	keys []map[string]types.AttributeValue,
) (any, error) {
	if batch {
		return c.batchGetItems(ctx, tableName, keys)
	}
	return c.getItem(ctx, tableName, key)
}

func (c *Client) GetItemAs(
	ctx context.Context,
	tableName string,
	key map[string]types.AttributeValue,
	batch bool,
	keys []map[string]types.AttributeValue,
	out any,
) error {
	if batch {
		return c.batchGetItemsAs(ctx, tableName, keys, out)
	}

	item, err := c.getItem(ctx, tableName, key)
	if err != nil {
		logger.Error("failed to get item", err)
		return WrapError(err, "GetItemAs get")
	}
	if err = attributevalue.UnmarshalMap(item, out); err != nil {
		logger.Error("failed to unmarshal item", err)
		return WrapError(err, "GetItemAs unmarshal")
	}
	return nil
}

func (c *Client) WriteItem(
	ctx context.Context,
	tableName string,
	item map[string]types.AttributeValue,
	batch bool,
	items []map[string]types.AttributeValue,
	condition *string,
) error {
	if batch {
		return c.batchWriteItems(ctx, tableName, items)
	}
	return c.putItem(ctx, tableName, item, condition)
}

func (c *Client) WriteItemFrom(
	ctx context.Context,
	tableName string,
	item any,
	batch bool,
	items any,
	condition *string,
) error {
	if batch {
		av, err := attributevalue.MarshalList(items)
		if err != nil {
			logger.Error("failed to marshal items", err)
			return WrapError(err, "WriteItemFrom marshal batch items")
		}

		avMaps := make([]map[string]types.AttributeValue, len(av))
		for i, it := range av {
			if m, ok := it.(*types.AttributeValueMemberM); ok {
				avMaps[i] = m.Value
			} else {
				return WrapError(fmt.Errorf("item %d is not a map", i), "WriteItemFrom marshal batch items")
			}
		}
		return c.batchWriteItems(ctx, tableName, avMaps)
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		logger.Error("failed to marshal item", err)
		return WrapError(err, "WriteItemFrom marshal single item")
	}
	return c.putItem(ctx, tableName, av, condition)
}

func (c *Client) UpdateItem(
	ctx context.Context,
	tableName string,
	key map[string]types.AttributeValue,
	updateExpr string,
	exprValues map[string]types.AttributeValue,
	exprNames map[string]string,
	condition *string,
) (map[string]types.AttributeValue, error) {
	input := &dynamodb.UpdateItemInput{
		TableName:        aws.String(tableName),
		Key:              key,
		UpdateExpression: aws.String(updateExpr),
		ReturnValues:     types.ReturnValueAllNew,
	}
	if exprValues != nil {
		input.ExpressionAttributeValues = exprValues
	}
	if exprNames != nil {
		input.ExpressionAttributeNames = exprNames
	}
	if condition != nil {
		input.ConditionExpression = condition
	}

	result, err := c.db.UpdateItem(ctx, input)
	if err != nil {
		logger.Error("failed to update item", err)
		return nil, WrapError(err, "UpdateItem")
	}
	return result.Attributes, nil
}

func (c *Client) UpdateItemAs(
	ctx context.Context,
	tableName string,
	key map[string]types.AttributeValue,
	updateExpr string,
	exprValues map[string]types.AttributeValue,
	exprNames map[string]string,
	condition *string,
	out any,
) error {
	attrs, err := c.UpdateItem(ctx, tableName, key, updateExpr, exprValues, exprNames, condition)
	if err != nil {
		logger.Error("failed to update item", err)
		return WrapError(err, "UpdateItemAs")
	}
	if err = attributevalue.UnmarshalMap(attrs, out); err != nil {
		logger.Error("failed to unmarshal item", err)
		return WrapError(err, "UpdateItemAs unmarshal")
	}
	return nil
}

func (c *Client) DeleteItem(
	ctx context.Context,
	tableName string,
	key map[string]types.AttributeValue,
	batch bool,
	keys []map[string]types.AttributeValue,
	condition *string,
) error {
	if batch {
		return c.batchDeleteItems(ctx, tableName, keys)
	}
	return c.deleteItem(ctx, tableName, key, condition)
}
