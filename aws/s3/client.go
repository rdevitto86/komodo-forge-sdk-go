package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	logger "github.com/rdevitto86/komodo-forge-sdk-go/logging/runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// API is the interface for S3 operations.
type API interface {
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
	GetObjectAs(ctx context.Context, bucket, key string, out any) error
	PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error
	DeleteObject(ctx context.Context, bucket, key string) error
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

// Client wraps the AWS S3 SDK client.
type Client struct {
	s3 *s3.Client
}

// New creates and returns a new S3 Client.
func New(config Config) (*Client, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("s3: region is required")
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

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), cfgOpts...)
	if err != nil {
		logger.Error("s3 failed to load config", err)
		return nil, WrapError(err, "New")
	}

	var opts []func(*s3.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(ep)
			o.UsePathStyle = true
		})
	}

	return &Client{s3: s3.NewFromConfig(cfg, opts...)}, nil
}

// GetObject retrieves an object from S3 as raw bytes.
func (c *Client) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	result, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		logger.Error("failed to get s3 object", err)
		return nil, WrapError(err, "GetObject")
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		logger.Error("failed to read s3 object body", err)
		return nil, WrapError(err, "GetObject read body")
	}
	return data, nil
}

// GetObjectAs retrieves an S3 object and unmarshals the JSON body into out.
func (c *Client) GetObjectAs(ctx context.Context, bucket, key string, out any) error {
	data, err := c.GetObject(ctx, bucket, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		logger.Error("failed to unmarshal s3 object", err)
		return WrapError(err, "GetObjectAs unmarshal")
	}
	return nil
}

// PutObject uploads data to S3 at bucket/key with an optional contentType.
func (c *Client) PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if _, err := c.s3.PutObject(ctx, input); err != nil {
		logger.Error("failed to put s3 object", err)
		return WrapError(err, "PutObject")
	}
	return nil
}

// DeleteObject removes an object from S3.
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	if _, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		logger.Error("failed to delete s3 object", err)
		return WrapError(err, "DeleteObject")
	}
	return nil
}
