package opensearch

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
	if err.Error() != "missing region" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestNew_ValidConfig(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestDescribeDomain_EmptyName(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	_, err = c.DescribeDomain(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty domain name, got nil")
	}
	if err.Error() != "domain name is required" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

// ── Integration Tests ────────────────────────────────────────────────────────

func TestLocalStack_ListDomainNames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	c, err := New(context.Background(), localstackConfig())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	names, err := c.ListDomainNames(context.Background())
	if err != nil {
		t.Fatalf("ListDomainNames returned error: %v", err)
	}
	t.Logf("found %d domain(s)", len(names))
}

func TestLocalStack_DescribeDomain_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	c, err := New(context.Background(), localstackConfig())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = c.DescribeDomain(context.Background(), "nonexistent-domain")
	if err == nil {
		t.Fatal("expected error for nonexistent domain, got nil")
	}
}
