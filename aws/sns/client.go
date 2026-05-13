package sns

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// PublishInput carries all parameters for an SNS publish call, including the
// optional FIFO fields GroupID and DedupID.
type PublishInput struct {
	TopicARN string
	Message  string
	GroupID  string            // FIFO MessageGroupId (required for .fifo topics)
	DedupID  string            // FIFO MessageDeduplicationId
	Attrs    map[string]string // optional string message attributes
}

type API interface {
	Publish(ctx context.Context, input PublishInput) (messageID string, err error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type Client struct {
	sns *sns.Client
}

// Creates and returns a new SNS Client.
func New(config Config) (*Client, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("region is required")
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
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	var opts []func(*sns.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *sns.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{sns: sns.NewFromConfig(cfg, opts...)}, nil
}

// Sends a message to an SNS topic. For FIFO topics, set GroupID and DedupID.
func (c *Client) Publish(ctx context.Context, input PublishInput) (string, error) {
	in := &sns.PublishInput{
		TopicArn: aws.String(input.TopicARN),
		Message:  aws.String(input.Message),
	}

	if input.GroupID != "" {
		in.MessageGroupId = aws.String(input.GroupID)
	}
	if input.DedupID != "" {
		in.MessageDeduplicationId = aws.String(input.DedupID)
	}

	if len(input.Attrs) > 0 {
		in.MessageAttributes = make(map[string]types.MessageAttributeValue, len(input.Attrs))
		for k, v := range input.Attrs {
			v := v
			in.MessageAttributes[k] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: &v,
			}
		}
	}

	result, err := c.sns.Publish(ctx, in)
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}
	return aws.ToString(result.MessageId), nil
}
