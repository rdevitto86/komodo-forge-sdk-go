package lambda

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type API interface {
	Invoke(ctx context.Context, functionName string, payload []byte) ([]byte, error)
	InvokeAsync(ctx context.Context, functionName string, payload []byte) error
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string // optional; set to LocalStack URL in non-prod environments
}

type Client struct {
	lambda *lambda.Client
}

// Creates a Lambda Client from the provided Config; returns an error if Region is empty.
func New(ctx context.Context, config Config) (*Client, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("missing region")
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
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	var opts []func(*lambda.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *lambda.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{lambda: lambda.NewFromConfig(cfg, opts...)}, nil
}

// Synchronously invokes the named Lambda function with payload and returns the response payload.
func (c *Client) Invoke(ctx context.Context, functionName string, payload []byte) ([]byte, error) {
	if functionName == "" {
		return nil, fmt.Errorf("missing function name")
	}

	result, err := c.lambda.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke function: %w", err)
	}
	return result.Payload, nil
}

// Fires a Lambda invocation event without waiting for a response (InvocationTypeEvent).
func (c *Client) InvokeAsync(ctx context.Context, functionName string, payload []byte) error {
	if functionName == "" {
		return fmt.Errorf("missing function name")
	}

	_, err := c.lambda.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: types.InvocationTypeEvent,
		Payload:        payload,
	})
	if err != nil {
		return fmt.Errorf("failed to invoke function async: %w", err)
	}
	return nil
}
