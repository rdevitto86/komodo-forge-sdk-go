package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamodb"
	"github.com/rdevitto86/komodo-forge-sdk-go/aws/s3"
	"github.com/rdevitto86/komodo-forge-sdk-go/db/redis"
)

type fakeDynamoDB struct {
	dynamodb.API
	table string
	err   error
}

func (f fakeDynamoDB) DescribeTable(ctx context.Context, table string) error {
	if table != f.table {
		return errors.New("unexpected table name")
	}
	return f.err
}

type fakeRedis struct {
	redis.API
	err error
}

func (f fakeRedis) Ping(ctx context.Context) error { return f.err }

type fakeS3 struct {
	s3.API
	bucket string
	err    error
}

func (f fakeS3) HeadBucket(ctx context.Context, bucket string) error {
	if bucket != f.bucket {
		return errors.New("unexpected bucket name")
	}
	return f.err
}

func TestDynamoDBChecker(t *testing.T) {
	healthy := DynamoDBChecker("orders-table", fakeDynamoDB{table: "orders"}, "orders")
	if err := healthy.Check(context.Background()); err != nil {
		t.Fatalf("expected healthy table to pass, got %v", err)
	}

	unhealthy := DynamoDBChecker("orders-table", fakeDynamoDB{table: "orders", err: errors.New("table not found")}, "orders")
	if err := unhealthy.Check(context.Background()); err == nil {
		t.Fatal("expected unreachable table to fail")
	}
}

func TestRedisChecker(t *testing.T) {
	healthy := RedisChecker("cache", fakeRedis{})
	if err := healthy.Check(context.Background()); err != nil {
		t.Fatalf("expected healthy redis to pass, got %v", err)
	}

	unhealthy := RedisChecker("cache", fakeRedis{err: errors.New("connection refused")})
	if err := unhealthy.Check(context.Background()); err == nil {
		t.Fatal("expected unreachable redis to fail")
	}
}

func TestS3Checker(t *testing.T) {
	healthy := S3Checker("uploads", fakeS3{bucket: "uploads"}, "uploads")
	if err := healthy.Check(context.Background()); err != nil {
		t.Fatalf("expected healthy bucket to pass, got %v", err)
	}

	unhealthy := S3Checker("uploads", fakeS3{bucket: "uploads", err: errors.New("access denied")}, "uploads")
	if err := unhealthy.Check(context.Background()); err == nil {
		t.Fatal("expected unreachable bucket to fail")
	}
}

func TestHTTPChecker(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()

	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()

	if err := HTTPChecker("upstream", ok.URL).Check(context.Background()); err != nil {
		t.Fatalf("expected 2xx response to pass, got %v", err)
	}
	if err := HTTPChecker("upstream", notFound.URL).Check(context.Background()); err == nil {
		t.Fatal("expected non-2xx response to fail")
	}
	if err := HTTPChecker("upstream", "http://127.0.0.1:0").Check(context.Background()); err == nil {
		t.Fatal("expected unreachable URL to fail")
	}
}
