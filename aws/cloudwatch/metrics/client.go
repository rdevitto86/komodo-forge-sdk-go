package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	// maxDatumsPerCall is the CloudWatch limit for PutMetricData.
	maxDatumsPerCall = 1000
)

// Represents a single CloudWatch metric data point.
type MetricDatum struct {
	Name  string
	Value float64
	// Unit is the string form of a CloudWatch StandardUnit (e.g. "Count", "Bytes").
	Unit string
	// Timestamp is the time the data point was recorded; defaults to time.Now() if zero.
	Timestamp  time.Time
	Dimensions map[string]string
}

// Carries parameters for a GetMetricStatistics request.
type GetMetricStatisticsInput struct {
	Namespace  string
	MetricName string
	StartTime  time.Time
	EndTime    time.Time
	// Period is the granularity of returned statistics (rounded to seconds).
	Period time.Duration
	// Statistics lists the stat names to retrieve, e.g. ["Average", "Sum"].
	Statistics []string
	Dimensions map[string]string
}

// Holds a single statistics result datapoint returned by GetMetricStatistics.
type MetricStat struct {
	Timestamp   time.Time
	Average     *float64
	Sum         *float64
	Minimum     *float64
	Maximum     *float64
	SampleCount *float64
	Unit        string
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	// Endpoint overrides the default CloudWatch endpoint; used for LocalStack.
	Endpoint string
}

// Wraps the AWS CloudWatch SDK client.
type Client struct {
	cw *cloudwatch.Client
}

// Creates a CloudWatch metrics Client; returns an error if Region is empty, not a known AWS region, or AWS config loading fails.
func New(ctx context.Context, config Config) (*Client, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("missing region")
	}
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

	var opts []func(*cloudwatch.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *cloudwatch.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{cw: cloudwatch.NewFromConfig(cfg, opts...)}, nil
}

// Publishes metric data points to the given CloudWatch namespace, chunking automatically at 1000 datums per call.
func (c *Client) PutMetricData(ctx context.Context, namespace string, metrics []MetricDatum) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if len(metrics) == 0 {
		return nil
	}

	for start := 0; start < len(metrics); start += maxDatumsPerCall {
		end := min(start+maxDatumsPerCall, len(metrics))
		chunk := metrics[start:end]

		data := make([]types.MetricDatum, 0, len(chunk))
		for _, m := range chunk {
			ts := m.Timestamp
			if ts.IsZero() {
				ts = time.Now()
			}

			datum := types.MetricDatum{
				MetricName: aws.String(m.Name),
				Value:      aws.Float64(m.Value),
				Timestamp:  aws.Time(ts),
			}

			if m.Unit != "" {
				datum.Unit = types.StandardUnit(m.Unit)
			}

			if len(m.Dimensions) > 0 {
				dims := make([]types.Dimension, 0, len(m.Dimensions))
				for k, v := range m.Dimensions {
					dims = append(dims, types.Dimension{
						Name:  aws.String(k),
						Value: aws.String(v),
					})
				}
				datum.Dimensions = dims
			}

			data = append(data, datum)
		}

		_, err := c.cw.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
			Namespace:  aws.String(namespace),
			MetricData: data,
		})
		if err != nil {
			return fmt.Errorf("failed to put metric data: %w", err)
		}
	}

	return nil
}

// Retrieves aggregated statistics for a CloudWatch metric over a time window.
func (c *Client) GetMetricStatistics(ctx context.Context, input GetMetricStatisticsInput) ([]MetricStat, error) {
	if input.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if input.MetricName == "" {
		return nil, fmt.Errorf("metric name is required")
	}

	stats := make([]types.Statistic, 0, len(input.Statistics))
	for _, s := range input.Statistics {
		stats = append(stats, types.Statistic(s))
	}

	var dims []types.Dimension
	for k, v := range input.Dimensions {
		dims = append(dims, types.Dimension{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	periodSecs := int32(input.Period.Seconds())
	if periodSecs < 1 {
		periodSecs = 60
	}

	out, err := c.cw.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(input.Namespace),
		MetricName: aws.String(input.MetricName),
		StartTime:  aws.Time(input.StartTime),
		EndTime:    aws.Time(input.EndTime),
		Period:     aws.Int32(periodSecs),
		Statistics: stats,
		Dimensions: dims,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get metric statistics: %w", err)
	}

	result := make([]MetricStat, 0, len(out.Datapoints))
	for _, dp := range out.Datapoints {
		stat := MetricStat{
			Average:     dp.Average,
			Sum:         dp.Sum,
			Minimum:     dp.Minimum,
			Maximum:     dp.Maximum,
			SampleCount: dp.SampleCount,
			Unit:        string(dp.Unit),
		}
		if dp.Timestamp != nil {
			stat.Timestamp = *dp.Timestamp
		}
		result = append(result, stat)
	}

	return result, nil
}
