package elasticache

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awselasticache "github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
)

type fakeElastiCache struct {
	rgPages []*awselasticache.DescribeReplicationGroupsOutput
	rgIdx   int
	ccPages []*awselasticache.DescribeCacheClustersOutput
	ccIdx   int
}

func (f *fakeElastiCache) DescribeReplicationGroups(ctx context.Context, params *awselasticache.DescribeReplicationGroupsInput, optFns ...func(*awselasticache.Options)) (*awselasticache.DescribeReplicationGroupsOutput, error) {
	out := f.rgPages[f.rgIdx]
	f.rgIdx++
	return out, nil
}

func (f *fakeElastiCache) DescribeCacheClusters(ctx context.Context, params *awselasticache.DescribeCacheClustersInput, optFns ...func(*awselasticache.Options)) (*awselasticache.DescribeCacheClustersOutput, error) {
	out := f.ccPages[f.ccIdx]
	f.ccIdx++
	return out, nil
}

func localstackConfig() Config {
	ep := os.Getenv("LOCALSTACK_ENDPOINT")
	if ep == "" {
		ep = "http://localhost:4566"
	}
	return Config{Region: "us-east-1", AccessKey: "test", SecretKey: "test", Endpoint: ep}
}

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

// ── Unit Tests ───────────────────────────────────────────────────────────────

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

func TestDescribeReplicationGroups_Paginates(t *testing.T) {
	marker := "m"
	f := &fakeElastiCache{
		rgPages: []*awselasticache.DescribeReplicationGroupsOutput{
			{ReplicationGroups: []ectypes.ReplicationGroup{{ReplicationGroupId: aws.String("a")}}, Marker: &marker},
			{ReplicationGroups: []ectypes.ReplicationGroup{{ReplicationGroupId: aws.String("b")}}},
		},
	}
	c := newWithAPI(f)
	got, err := c.DescribeReplicationGroups(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 groups across pages, got %d", len(got))
	}
}

func TestDescribeCacheClusters_Paginates(t *testing.T) {
	marker := "m"
	f := &fakeElastiCache{
		ccPages: []*awselasticache.DescribeCacheClustersOutput{
			{CacheClusters: []ectypes.CacheCluster{{CacheClusterId: aws.String("a")}}, Marker: &marker},
			{CacheClusters: []ectypes.CacheCluster{{CacheClusterId: aws.String("b")}}},
		},
	}
	c := newWithAPI(f)
	got, err := c.DescribeCacheClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 clusters across pages, got %d", len(got))
	}
}

// ── Integration Tests ────────────────────────────────────────────────────────

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
