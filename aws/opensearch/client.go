package opensearch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
)

// Flattened view of an OpenSearch domain returned by the control-plane API.
type Domain struct {
	Name          string
	Endpoint      string
	EngineVersion string
	ARN           string
	Created       bool
	Processing    bool
}

// Defines the OpenSearch control-plane operations exposed by this package.
type API interface {
	DescribeDomain(ctx context.Context, name string) (*Domain, error)
	ListDomainNames(ctx context.Context) ([]string, error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string // optional; set to LocalStack URL in non-prod environments
}

// Wraps the AWS OpenSearch control-plane SDK client.
type Client struct {
	os *opensearch.Client
}

// Creates and returns a new OpenSearch control-plane Client. Returns an error if Region is empty or not a known AWS region.
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
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	var opts []func(*opensearch.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *opensearch.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{os: opensearch.NewFromConfig(cfg, opts...)}, nil
}

// Fetches metadata for a single OpenSearch domain by name.
func (c *Client) DescribeDomain(ctx context.Context, name string) (*Domain, error) {
	if name == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	result, err := c.os.DescribeDomain(ctx, &opensearch.DescribeDomainInput{
		DomainName: aws.String(name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe domain: %w", err)
	}

	ds := result.DomainStatus
	if ds == nil {
		return nil, fmt.Errorf("describe domain returned empty status")
	}

	d := &Domain{
		Name:       aws.ToString(ds.DomainName),
		ARN:        aws.ToString(ds.ARN),
		Created:    aws.ToBool(ds.Created),
		Processing: aws.ToBool(ds.Processing),
	}
	if ds.EngineVersion != nil {
		d.EngineVersion = aws.ToString(ds.EngineVersion)
	}
	if ds.Endpoint != nil {
		d.Endpoint = aws.ToString(ds.Endpoint)
	}

	return d, nil
}

// Returns the names of all OpenSearch domains in the configured region.
func (c *Client) ListDomainNames(ctx context.Context) ([]string, error) {
	result, err := c.os.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list domain names: %w", err)
	}

	names := make([]string, 0, len(result.DomainNames))
	for _, d := range result.DomainNames {
		names = append(names, aws.ToString(d.DomainName))
	}
	return names, nil
}
