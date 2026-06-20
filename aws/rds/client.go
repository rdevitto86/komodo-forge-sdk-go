package rds

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

type API interface {
	ExecuteStatement(ctx context.Context, input ExecuteStatementInput) (*ExecuteStatementOutput, error)
	BatchExecuteStatement(ctx context.Context, input BatchExecuteStatementInput) (*BatchExecuteStatementOutput, error)
	BeginTransaction(ctx context.Context) (transactionID string, err error)
	CommitTransaction(ctx context.Context, transactionID string) error
	RollbackTransaction(ctx context.Context, transactionID string) error
}

type Config struct {
	Region      string
	AccessKey   string
	SecretKey   string
	Endpoint    string
	ResourceArn string
	SecretArn   string
	Database    string
}

type ExecuteStatementInput struct {
	SQL           string
	Parameters    map[string]any
	TransactionID string // optional
	Database      string // optional
}

type ExecuteStatementOutput struct {
	Rows            []map[string]any
	NumRecords      int64
	GeneratedFields []any
}

type BatchExecuteStatementInput struct {
	SQL           string
	ParameterSets []map[string]any
	TransactionID string
	Database      string
}

type BatchExecuteStatementOutput struct {
	UpdateResults []BatchUpdateResult
}

type BatchUpdateResult struct {
	GeneratedFields []any
}

type rdsDataAPI interface {
	ExecuteStatement(ctx context.Context, in *rdsdata.ExecuteStatementInput, opts ...func(*rdsdata.Options)) (*rdsdata.ExecuteStatementOutput, error)
	BatchExecuteStatement(ctx context.Context, in *rdsdata.BatchExecuteStatementInput, opts ...func(*rdsdata.Options)) (*rdsdata.BatchExecuteStatementOutput, error)
	BeginTransaction(ctx context.Context, in *rdsdata.BeginTransactionInput, opts ...func(*rdsdata.Options)) (*rdsdata.BeginTransactionOutput, error)
	CommitTransaction(ctx context.Context, in *rdsdata.CommitTransactionInput, opts ...func(*rdsdata.Options)) (*rdsdata.CommitTransactionOutput, error)
	RollbackTransaction(ctx context.Context, in *rdsdata.RollbackTransactionInput, opts ...func(*rdsdata.Options)) (*rdsdata.RollbackTransactionOutput, error)
}

type Client struct {
	api         rdsDataAPI
	resourceARN string
	secretARN   string
	database    string
}

func New(ctx context.Context, config Config) (*Client, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("missing region")
	}
	if config.ResourceArn == "" {
		return nil, fmt.Errorf("missing ResourceArn")
	}
	if config.SecretArn == "" {
		return nil, fmt.Errorf("missing SecretArn")
	}

	cfgOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(config.Region),
	}

	if config.AccessKey != "" && config.SecretKey != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, ""),
		))
	} else if config.Endpoint != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var opts []func(*rdsdata.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *rdsdata.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{
		api:         rdsdata.NewFromConfig(cfg, opts...),
		resourceARN: config.ResourceArn,
		secretARN:   config.SecretArn,
		database:    config.Database,
	}, nil
}

func newWithAPI(api rdsDataAPI, resourceARN, secretARN, database string) *Client {
	return &Client{
		api:         api,
		resourceARN: resourceARN,
		secretARN:   secretARN,
		database:    database,
	}
}

func (c *Client) ExecuteStatement(ctx context.Context, input ExecuteStatementInput) (*ExecuteStatementOutput, error) {
	if input.SQL == "" {
		return nil, fmt.Errorf("SQL is required")
	}

	params, err := buildSQLParameters(input.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL parameters: %w", err)
	}

	db := c.database
	if input.Database != "" {
		db = input.Database
	}

	in := &rdsdata.ExecuteStatementInput{
		ResourceArn:           aws.String(c.resourceARN),
		SecretArn:             aws.String(c.secretARN),
		Sql:                   aws.String(input.SQL),
		IncludeResultMetadata: true,
		Parameters:            params,
	}

	if db != "" {
		in.Database = aws.String(db)
	}
	if input.TransactionID != "" {
		in.TransactionId = aws.String(input.TransactionID)
	}

	out, err := c.api.ExecuteStatement(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement: %w", err)
	}

	rows, err := decodeRecords(out.Records, out.ColumnMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result rows: %w", err)
	}

	generatedFields, err := decodeFields(out.GeneratedFields)
	if err != nil {
		return nil, fmt.Errorf("failed to decode generated fields: %w", err)
	}

	return &ExecuteStatementOutput{
		Rows:            rows,
		NumRecords:      out.NumberOfRecordsUpdated,
		GeneratedFields: generatedFields,
	}, nil
}

func (c *Client) BatchExecuteStatement(ctx context.Context, input BatchExecuteStatementInput) (*BatchExecuteStatementOutput, error) {
	if input.SQL == "" {
		return nil, fmt.Errorf("SQL is required")
	}

	paramSets := make([][]types.SqlParameter, len(input.ParameterSets))
	for i, ps := range input.ParameterSets {
		built, err := buildSQLParameters(ps)
		if err != nil {
			return nil, fmt.Errorf("failed to build SQL parameters for set %d: %w", i, err)
		}
		paramSets[i] = built
	}

	db := c.database
	if input.Database != "" {
		db = input.Database
	}

	in := &rdsdata.BatchExecuteStatementInput{
		ResourceArn:   aws.String(c.resourceARN),
		SecretArn:     aws.String(c.secretARN),
		Sql:           aws.String(input.SQL),
		ParameterSets: paramSets,
	}

	if db != "" {
		in.Database = aws.String(db)
	}
	if input.TransactionID != "" {
		in.TransactionId = aws.String(input.TransactionID)
	}

	out, err := c.api.BatchExecuteStatement(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to batch execute statement: %w", err)
	}

	results := make([]BatchUpdateResult, len(out.UpdateResults))
	for i, ur := range out.UpdateResults {
		gf, err := decodeFields(ur.GeneratedFields)
		if err != nil {
			return nil, fmt.Errorf("failed to decode generated fields for update result %d: %w", i, err)
		}
		results[i] = BatchUpdateResult{GeneratedFields: gf}
	}

	return &BatchExecuteStatementOutput{UpdateResults: results}, nil
}

func (c *Client) BeginTransaction(ctx context.Context) (string, error) {
	in := &rdsdata.BeginTransactionInput{
		ResourceArn: aws.String(c.resourceARN),
		SecretArn:   aws.String(c.secretARN),
	}

	if c.database != "" {
		in.Database = aws.String(c.database)
	}

	out, err := c.api.BeginTransaction(ctx, in)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	return aws.ToString(out.TransactionId), nil
}

func (c *Client) CommitTransaction(ctx context.Context, transactionID string) error {
	_, err := c.api.CommitTransaction(ctx, &rdsdata.CommitTransactionInput{
		ResourceArn:   aws.String(c.resourceARN),
		SecretArn:     aws.String(c.secretARN),
		TransactionId: aws.String(transactionID),
	})
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (c *Client) RollbackTransaction(ctx context.Context, transactionID string) error {
	_, err := c.api.RollbackTransaction(ctx, &rdsdata.RollbackTransactionInput{
		ResourceArn:   aws.String(c.resourceARN),
		SecretArn:     aws.String(c.secretARN),
		TransactionId: aws.String(transactionID),
	})
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

func buildSQLParameters(params map[string]any) ([]types.SqlParameter, error) {
	if len(params) == 0 {
		return nil, nil
	}

	out := make([]types.SqlParameter, 0, len(params))
	for name, val := range params {
		f, err := toField(val)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter %q: %w", name, err)
		}
		n := name
		out = append(out, types.SqlParameter{Name: &n, Value: f})
	}
	return out, nil
}

func decodeRecords(records [][]types.Field, cols []types.ColumnMetadata) ([]map[string]any, error) {
	if len(records) == 0 {
		return nil, nil
	}

	rows := make([]map[string]any, len(records))
	for i, record := range records {
		row := make(map[string]any, len(record))
		for j, field := range record {
			colName := ""
			if j < len(cols) {
				colName = aws.ToString(cols[j].Name)
			}
			if colName == "" {
				colName = fmt.Sprintf("col_%d", j)
			}
			val, err := fromField(field)
			if err != nil {
				return nil, fmt.Errorf("failed to decode field %q in row %d: %w", colName, i, err)
			}
			row[colName] = val
		}
		rows[i] = row
	}
	return rows, nil
}

func decodeFields(fields []types.Field) ([]any, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	out := make([]any, len(fields))
	for i, f := range fields {
		val, err := fromField(f)
		if err != nil {
			return nil, fmt.Errorf("failed to decode field at index %d: %w", i, err)
		}
		out[i] = val
	}
	return out, nil
}

var _ API = (*Client)(nil)
