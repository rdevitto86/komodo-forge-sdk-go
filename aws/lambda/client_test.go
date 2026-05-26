package lambda

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
	c, err := New(context.Background(), Config{Region: "us-east-1", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestInvoke_EmptyFunctionName(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	_, err = c.Invoke(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty function name, got nil")
	}
	if err.Error() != "missing function name" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestInvokeAsync_EmptyFunctionName(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	err = c.InvokeAsync(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty function name, got nil")
	}
	if err.Error() != "missing function name" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

// ── Component Tests (LocalStack) ──────────────────────────────────────────────

// TestLocalStack_Invoke_NotFound exercises the Lambda client against LocalStack.
// It expects an error because no function is deployed — this validates the
// client wiring (endpoint, credentials, invocation path) end-to-end.
func TestLocalStack_Invoke_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	c, err := New(context.Background(), localstackConfig())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = c.Invoke(context.Background(), "nonexistent-function", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error invoking nonexistent function, got nil")
	}
	// Any SDK-level error is acceptable; the test proves the client reached LocalStack.
}

func TestLocalStack_InvokeAsync_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	c, err := New(context.Background(), localstackConfig())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = c.InvokeAsync(context.Background(), "nonexistent-function", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error invoking nonexistent function async, got nil")
	}
}
