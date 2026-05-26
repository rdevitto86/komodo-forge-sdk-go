package metrics

import (
	"context"
	"net"
	"os"
	"testing"
	"time"
)

func localstackConfig() Config {
	ep := os.Getenv("LOCALSTACK_ENDPOINT")
	if ep == "" {
		ep = "http://localhost:4566"
	}
	return Config{Region: "us-east-1", AccessKey: "test", SecretKey: "test", Endpoint: ep}
}

// checkLocalstack skips the test if running in short mode or if LocalStack is unreachable.
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

// ── Unit Tests ─────────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
}

func TestPutMetricData_EmptyNamespace(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	err = c.PutMetricData(context.Background(), "", []MetricDatum{{Name: "Foo", Value: 1}})
	if err == nil {
		t.Fatal("expected error for empty namespace, got nil")
	}
}

func TestGetMetricStatistics_EmptyNamespace(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	_, err = c.GetMetricStatistics(context.Background(), GetMetricStatisticsInput{
		MetricName: "Foo",
	})
	if err == nil {
		t.Fatal("expected error for empty namespace, got nil")
	}
}

// ── Component Tests (LocalStack) ───────────────────────────────────────────────

func TestPutAndGetMetricStatistics_LocalStack(t *testing.T) {
	checkLocalstack(t)

	cfg := localstackConfig()
	c, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	now := time.Now().UTC()
	datums := []MetricDatum{
		{Name: "RequestCount", Value: 10, Unit: "Count", Timestamp: now},
		{Name: "RequestCount", Value: 20, Unit: "Count", Timestamp: now},
		{Name: "RequestCount", Value: 30, Unit: "Count", Timestamp: now},
	}

	if err := c.PutMetricData(context.Background(), "Komodo/Test", datums); err != nil {
		t.Fatalf("failed to put metric data: %v", err)
	}

	stats, err := c.GetMetricStatistics(context.Background(), GetMetricStatisticsInput{
		Namespace:  "Komodo/Test",
		MetricName: "RequestCount",
		StartTime:  now.Add(-60 * time.Second),
		EndTime:    now.Add(60 * time.Second),
		Period:     60 * time.Second,
		Statistics: []string{"Sum", "Average"},
	})
	if err != nil {
		t.Fatalf("failed to get metric statistics: %v", err)
	}
	if len(stats) == 0 {
		t.Fatal("expected at least one stat datapoint, got none")
	}
}
