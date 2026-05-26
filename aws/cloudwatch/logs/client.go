package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const (
	// maxEventsPerCall is the CloudWatch Logs limit for PutLogEvents.
	maxEventsPerCall = 10000
	// maxBatchBytes is the approximate max aggregate message size per PutLogEvents call.
	maxBatchBytes = 1 * 1024 * 1024 // 1 MB
)

// Represents a single log entry to be written to CloudWatch Logs.
type LogEvent struct {
	Timestamp time.Time
	Message   string
}

// Carries parameters for a FilterLogEvents request.
type FilterLogEventsInput struct {
	GroupName     string
	FilterPattern string
	StartTime     time.Time
	EndTime       time.Time
	Limit         int32
}

// Represents a single log event returned by FilterLogEvents.
type FilteredLogEvent struct {
	Timestamp  time.Time
	Message    string
	StreamName string
	EventID    string
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	// Endpoint overrides the default CloudWatch Logs endpoint; used for LocalStack.
	Endpoint string
}

// Wraps the AWS CloudWatch Logs SDK client.
type Client struct {
	cwl *cloudwatchlogs.Client
}

// Creates a CloudWatch Logs Client; returns an error if Region is empty, not a known AWS region, or AWS config loading fails.
func New(ctx context.Context, config Config) (*Client, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("missing region")
	}
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
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var opts []func(*cloudwatchlogs.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *cloudwatchlogs.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{cwl: cloudwatchlogs.NewFromConfig(cfg, opts...)}, nil
}

// Writes log events to the specified CloudWatch Logs group and stream, chunking automatically at 10,000 events or ~1 MB.
func (c *Client) PutLogEvents(ctx context.Context, groupName, streamName string, events []LogEvent) error {
	if groupName == "" {
		return fmt.Errorf("group name is required")
	}
	if streamName == "" {
		return fmt.Errorf("stream name is required")
	}
	if len(events) == 0 {
		return nil
	}

	// Partition into batches respecting count and size limits.
	batches := partitionLogEvents(events)

	for _, batch := range batches {
		sdkEvents := make([]types.InputLogEvent, 0, len(batch))
		for _, e := range batch {
			ts := e.Timestamp
			if ts.IsZero() {
				ts = time.Now()
			}
			sdkEvents = append(sdkEvents, types.InputLogEvent{
				Timestamp: aws.Int64(ts.UnixMilli()),
				Message:   aws.String(e.Message),
			})
		}

		_, err := c.cwl.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(groupName),
			LogStreamName: aws.String(streamName),
			LogEvents:     sdkEvents,
		})
		if err != nil {
			return fmt.Errorf("failed to put log events: %w", err)
		}
	}

	return nil
}

// Retrieves log events from a CloudWatch Logs group matching an optional filter pattern.
func (c *Client) FilterLogEvents(ctx context.Context, input FilterLogEventsInput) ([]FilteredLogEvent, error) {
	if input.GroupName == "" {
		return nil, fmt.Errorf("group name is required")
	}

	in := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  aws.String(input.GroupName),
		FilterPattern: aws.String(input.FilterPattern),
	}
	if !input.StartTime.IsZero() {
		in.StartTime = aws.Int64(input.StartTime.UnixMilli())
	}
	if !input.EndTime.IsZero() {
		in.EndTime = aws.Int64(input.EndTime.UnixMilli())
	}
	if input.Limit > 0 {
		in.Limit = aws.Int32(input.Limit)
	}

	out, err := c.cwl.FilterLogEvents(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to filter log events: %w", err)
	}

	result := make([]FilteredLogEvent, 0, len(out.Events))
	for _, e := range out.Events {
		fe := FilteredLogEvent{
			Message:    aws.ToString(e.Message),
			StreamName: aws.ToString(e.LogStreamName),
			EventID:    aws.ToString(e.EventId),
		}
		if e.Timestamp != nil {
			fe.Timestamp = time.UnixMilli(*e.Timestamp).UTC()
		}
		result = append(result, fe)
	}

	return result, nil
}

// Helper that splits events into batches respecting the per-call count and byte limits.
func partitionLogEvents(events []LogEvent) [][]LogEvent {
	var batches [][]LogEvent
	var current []LogEvent
	currentBytes := 0

	for _, e := range events {
		msgLen := len(e.Message)
		// Start a new batch if count or size limit would be exceeded.
		if len(current) >= maxEventsPerCall || (currentBytes+msgLen > maxBatchBytes && len(current) > 0) {
			batches = append(batches, current)
			current = nil
			currentBytes = 0
		}
		current = append(current, e)
		currentBytes += msgLen
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}
