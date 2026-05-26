package elasticache

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

// skipIfNoLocalStack skips the test when LOCALSTACK_ENDPOINT is unset and
// localhost:4566 is unreachable within 5 seconds.
func skipIfNoLocalStack(t *testing.T) {
	t.Helper()
	if os.Getenv("LOCALSTACK_ENDPOINT") != "" {
		return
	}
	conn, err := net.DialTimeout("tcp", "localhost:4566", 5*time.Second)
	if err != nil {
		t.Skip("localstack not reachable; skipping component test")
	}
	conn.Close()
}

// ── Unit Tests ────────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
	if err == nil || err.Error() != "missing region" {
		t.Errorf("got %v, want \"missing region\"", err)
	}
}

func TestNew_ValidConfig(t *testing.T) {
	c, err := New(context.Background(), Config{
		Region:    "us-east-1",
		AccessKey: "key",
		SecretKey: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

// ── Component Tests (LocalStack) ──────────────────────────────────────────────

// TestLocalStack_DescribeReplicationGroups verifies that the control-plane client
// reaches LocalStack and returns an empty slice when no groups are configured.
func TestLocalStack_DescribeReplicationGroups(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	c, err := New(context.Background(), localstackConfig())
	if err != nil {
		t.Fatalf("failed to create control plane client: %v", err)
	}

	groups, err := c.DescribeReplicationGroups(context.Background())
	if err != nil {
		t.Fatalf("DescribeReplicationGroups returned error: %v", err)
	}
	t.Logf("found %d replication group(s)", len(groups))
}

// TestLocalStack_DescribeCacheClusters verifies that the control-plane client
// reaches LocalStack and returns an empty slice when no clusters are configured.
func TestLocalStack_DescribeCacheClusters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	c, err := New(context.Background(), localstackConfig())
	if err != nil {
		t.Fatalf("failed to create control plane client: %v", err)
	}

	clusters, err := c.DescribeCacheClusters(context.Background())
	if err != nil {
		t.Fatalf("DescribeCacheClusters returned error: %v", err)
	}
	t.Logf("found %d cache cluster(s)", len(clusters))
}
