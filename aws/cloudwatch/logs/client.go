package logs

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const (
	maxEventsPerCall = 10000
	maxBatchBytes    = 1 * 1024 * 1024
	logEventOverhead = 26
)

type LogEvent struct {
	Timestamp time.Time
	Message   string
}

type FilterLogEventsInput struct {
	GroupName     string
	FilterPattern string
	StartTime     time.Time
	EndTime       time.Time
	Limit         int32
}

type FilteredLogEvent struct {
	Timestamp  time.Time
	Message    string
	StreamName string
	EventID    string
}

type cwLogsAPI interface {
	PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error)
	FilterLogEvents(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type Client struct {
	cwl cwLogsAPI
}

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
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var opts []func(*cloudwatchlogs.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *cloudwatchlogs.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{cwl: cloudwatchlogs.NewFromConfig(cfg, opts...)}, nil
}

func newWithAPI(api cwLogsAPI) *Client {
	return &Client{cwl: api}
}

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

	resolved := make([]LogEvent, len(events))
	for i, e := range events {
		if e.Timestamp.IsZero() {
			e.Timestamp = time.Now()
		}
		resolved[i] = e
	}
	sort.SliceStable(resolved, func(i, j int) bool {
		return resolved[i].Timestamp.Before(resolved[j].Timestamp)
	})

	for _, batch := range partitionLogEvents(resolved) {
		sdkEvents := make([]types.InputLogEvent, 0, len(batch))
		for _, e := range batch {
			sdkEvents = append(sdkEvents, types.InputLogEvent{
				Timestamp: aws.Int64(e.Timestamp.UnixMilli()),
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

	var result []FilteredLogEvent
	for {
		out, err := c.cwl.FilterLogEvents(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("failed to filter log events: %w", err)
		}

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
			if input.Limit > 0 && int32(len(result)) >= input.Limit {
				return result[:input.Limit], nil
			}
		}

		if out.NextToken == nil {
			break
		}
		in.NextToken = out.NextToken
	}

	return result, nil
}

func partitionLogEvents(events []LogEvent) [][]LogEvent {
	var batches [][]LogEvent
	var current []LogEvent
	currentBytes := 0

	for _, e := range events {
		msgSize := len(e.Message) + logEventOverhead
		if len(current) >= maxEventsPerCall || (currentBytes+msgSize > maxBatchBytes && len(current) > 0) {
			batches = append(batches, current)
			current = nil
			currentBytes = 0
		}
		current = append(current, e)
		currentBytes += msgSize
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}
