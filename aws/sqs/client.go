package sqs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SendInput carries all parameters for an SQS send call, including optional
// FIFO fields GroupID and DedupID.
type SendInput struct {
	QueueURL string
	Body     string
	GroupID  string            // FIFO MessageGroupId (required for .fifo queues)
	DedupID  string            // FIFO MessageDeduplicationId
	Attrs    map[string]string // optional string message attributes
}

type Message struct {
	ID            string
	Body          string
	ReceiptHandle string
	Attrs         map[string]string
}

type API interface {
	Send(ctx context.Context, input SendInput) (messageID string, err error)
	Receive(ctx context.Context, queueURL string, maxMessages int32) ([]Message, error)
	Delete(ctx context.Context, queueURL, receiptHandle string) error
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type Client struct {
	sqs *sqs.Client
}

// Creates and returns a new SQS Client.
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

	var opts []func(*sqs.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *sqs.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{sqs: sqs.NewFromConfig(cfg, opts...)}, nil
}

// Enqueues a message. For FIFO queues, set GroupID and DedupID.
func (c *Client) Send(ctx context.Context, input SendInput) (string, error) {
	in := &sqs.SendMessageInput{
		QueueUrl:    aws.String(input.QueueURL),
		MessageBody: aws.String(input.Body),
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

	result, err := c.sqs.SendMessage(ctx, in)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	return aws.ToString(result.MessageId), nil
}

// Long-polls for up to maxMessages (max 10) from the queue.
func (c *Client) Receive(ctx context.Context, queueURL string, maxMessages int32) ([]Message, error) {
	if maxMessages > 10 {
		maxMessages = 10
	}

	result, err := c.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(queueURL),
		MaxNumberOfMessages:   maxMessages,
		WaitTimeSeconds:       20, // long-poll
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to receive message: %w", err)
	}

	msgs := make([]Message, len(result.Messages))
	for i, m := range result.Messages {
		attrs := make(map[string]string, len(m.MessageAttributes))
		for k, v := range m.MessageAttributes {
			if v.StringValue != nil {
				attrs[k] = *v.StringValue
			}
		}
		msgs[i] = Message{
			ID:            aws.ToString(m.MessageId),
			Body:          aws.ToString(m.Body),
			ReceiptHandle: aws.ToString(m.ReceiptHandle),
			Attrs:         attrs,
		}
	}
	return msgs, nil
}

// Removes a message from the queue after successful processing.
func (c *Client) Delete(ctx context.Context, queueURL, receiptHandle string) error {
	_, err := c.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}
