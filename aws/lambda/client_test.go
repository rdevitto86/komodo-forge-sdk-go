package lambda

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type fakeLambda struct {
	out *lambda.InvokeOutput
	err error
}

func (f *fakeLambda) Invoke(ctx context.Context, params *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	return f.out, f.err
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

func TestInvoke_FunctionError(t *testing.T) {
	c := newWithAPI(&fakeLambda{out: &lambda.InvokeOutput{
		FunctionError: aws.String("Unhandled"),
		Payload:       []byte(`{"errorMessage":"boom"}`),
	}})
	_, err := c.Invoke(context.Background(), "fn", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error when FunctionError is set, got nil")
	}
}

func TestInvoke_Success(t *testing.T) {
	want := []byte(`{"ok":true}`)
	c := newWithAPI(&fakeLambda{out: &lambda.InvokeOutput{Payload: want}})
	got, err := c.Invoke(context.Background(), "fn", []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %s, want %s", got, want)
	}
}

// ── Integration Tests ────────────────────────────────────────────────────────

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
