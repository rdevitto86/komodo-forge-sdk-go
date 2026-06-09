package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rdevitto86/komodo-forge-sdk-go/aws/dynamodb"
	"github.com/rdevitto86/komodo-forge-sdk-go/aws/s3"
	"github.com/rdevitto86/komodo-forge-sdk-go/db/redis"
)

const httpCheckTimeout = 2 * time.Second

// Reports whether the named table is reachable via DescribeTable.
func DynamoDBChecker(name string, client dynamodb.API, table string) Checker {
	return CheckerFunc(name, func(ctx context.Context) error {
		return client.DescribeTable(ctx, table)
	})
}

// Reports whether the Redis server responds to a Ping.
func RedisChecker(name string, client redis.API) Checker {
	return CheckerFunc(name, func(ctx context.Context) error {
		return client.Ping(ctx)
	})
}

// Reports whether the named bucket is reachable via HeadBucket.
func S3Checker(name string, client s3.API, bucket string) Checker {
	return CheckerFunc(name, func(ctx context.Context) error {
		return client.HeadBucket(ctx, bucket)
	})
}

// Reports whether a GET against url returns a 2xx status within 2s.
func HTTPChecker(name, url string) Checker {
	return CheckerFunc(name, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, httpCheckTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to build health check request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to reach %s: %w", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("received non-2xx status %d from %s", resp.StatusCode, url)
		}
		return nil
	})
}
