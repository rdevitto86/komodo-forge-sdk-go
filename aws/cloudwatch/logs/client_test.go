package logs

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func localstackConfig() Config {
	ep := os.Getenv("LOCALSTACK_ENDPOINT")
	if ep == "" {
		ep = "http://localhost:4566"
	}
	return Config{Region: "us-east-1", AccessKey: "test", SecretKey: "test", Endpoint: ep}
}

func checkLocalstack(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping LocalStack test in short mode")
	}
	conn, err := net.DialTimeout("tcp", "localhost:4566", 5*time.Second)
	if err != nil {
		t.Skip("LocalStack not reachable, skipping component test")
	}
	conn.Close()
}

func createLogGroupAndStream(t *testing.T, cfg Config, groupName, streamName string) {
	t.Helper()

	cfgOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), cfgOpts...)
	if err != nil {
		t.Fatalf("failed to load AWS config for test setup: %v", err)
	}

	ep := cfg.Endpoint
	cwlClient := cloudwatchlogs.NewFromConfig(awsCfg, func(o *cloudwatchlogs.Options) {
		o.BaseEndpoint = aws.String(ep)
	})

	// Create group — ignore AlreadyExistsException.
	_, err = cwlClient.CreateLogGroup(context.Background(), &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(groupName),
	})
	if err != nil {
		// AlreadyExists is acceptable; any other error is fatal.
		t.Logf("CreateLogGroup: %v (may already exist)", err)
	}

	// Create stream — ignore AlreadyExistsException.
	_, err = cwlClient.CreateLogStream(context.Background(), &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(groupName),
		LogStreamName: aws.String(streamName),
	})
	if err != nil {
		t.Logf("CreateLogStream: %v (may already exist)", err)
	}
}

// ── Unit Tests ───────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
}

func TestPutLogEvents_EmptyGroupName(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	err = c.PutLogEvents(context.Background(), "", "stream", []LogEvent{{Message: "hi"}})
	if err == nil {
		t.Fatal("expected error for empty group name, got nil")
	}
}

func TestFilterLogEvents_EmptyGroupName(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	_, err = c.FilterLogEvents(context.Background(), FilterLogEventsInput{})
	if err == nil {
		t.Fatal("expected error for empty group name, got nil")
	}
}

func TestPartitionLogEvents_ChunksAtLimit(t *testing.T) {
	events := make([]LogEvent, maxEventsPerCall+1)
	for i := range events {
		events[i] = LogEvent{Message: fmt.Sprintf("msg %d", i)}
	}
	batches := partitionLogEvents(events)
	if len(batches) != 2 {
		t.Fatalf("expected 2 batches for %d events, got %d", maxEventsPerCall+1, len(batches))
	}
	if len(batches[0]) != maxEventsPerCall {
		t.Fatalf("expected first batch to have %d events, got %d", maxEventsPerCall, len(batches[0]))
	}
}

// ── Integration Tests ────────────────────────────────────────────────────────

func TestPutAndFilterLogEvents_LocalStack(t *testing.T) {
	checkLocalstack(t)

	cfg := localstackConfig()
	groupName := fmt.Sprintf("komodo-test-%d", time.Now().UnixNano())
	streamName := "test-stream"

	createLogGroupAndStream(t, cfg, groupName, streamName)

	c, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	needle := fmt.Sprintf("unique-sentinel-%d", time.Now().UnixNano())
	events := []LogEvent{
		{Timestamp: time.Now(), Message: "unrelated log line"},
		{Timestamp: time.Now(), Message: needle},
		{Timestamp: time.Now(), Message: "another unrelated line"},
	}

	if err := c.PutLogEvents(context.Background(), groupName, streamName, events); err != nil {
		t.Fatalf("failed to put log events: %v", err)
	}

	filtered, err := c.FilterLogEvents(context.Background(), FilterLogEventsInput{
		GroupName:     groupName,
		FilterPattern: needle,
		StartTime:     time.Now().Add(-5 * time.Minute),
		EndTime:       time.Now().Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("failed to filter log events: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatalf("expected at least one filtered event matching %q, got none", needle)
	}
}
