package elasticache

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awselasticache "github.com/aws/aws-sdk-go-v2/service/elasticache"
)

// Flattens an ElastiCache replication group.
type ReplicationGroup struct {
	ID            string
	Status        string
	NodeType      string
	NumNodeGroups int32
	Endpoint      string
}

// Flattens an ElastiCache cache cluster.
type CacheCluster struct {
	ID            string
	Status        string
	NodeType      string
	Engine        string
	EngineVersion string
	NumCacheNodes int32
}

type API interface {
	DescribeReplicationGroups(ctx context.Context) ([]ReplicationGroup, error)
	DescribeCacheClusters(ctx context.Context) ([]CacheCluster, error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string // optional; set to LocalStack URL in non-prod environments
}

type Client struct {
	ec *awselasticache.Client
}

// Creates an ElastiCache control-plane Client; returns an error if Region is empty.
func New(ctx context.Context, config Config) (*Client, error) {
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

	var opts []func(*awselasticache.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *awselasticache.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{ec: awselasticache.NewFromConfig(cfg, opts...)}, nil
}

// Lists all ElastiCache replication groups in the configured region, returning a flattened slice with ID, status, node type, shard count, and primary endpoint.
func (c *Client) DescribeReplicationGroups(ctx context.Context) ([]ReplicationGroup, error) {
	result, err := c.ec.DescribeReplicationGroups(ctx, &awselasticache.DescribeReplicationGroupsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe replication groups: %w", err)
	}

	groups := make([]ReplicationGroup, 0, len(result.ReplicationGroups))
	for _, rg := range result.ReplicationGroups {
		g := ReplicationGroup{
			ID:            aws.ToString(rg.ReplicationGroupId),
			Status:        aws.ToString(rg.Status),
			NodeType:      aws.ToString(rg.CacheNodeType),
			NumNodeGroups: int32(len(rg.NodeGroups)),
		}
		if rg.ConfigurationEndpoint != nil {
			g.Endpoint = aws.ToString(rg.ConfigurationEndpoint.Address)
		} else if len(rg.NodeGroups) > 0 && rg.NodeGroups[0].PrimaryEndpoint != nil {
			g.Endpoint = aws.ToString(rg.NodeGroups[0].PrimaryEndpoint.Address)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// Lists all ElastiCache cache clusters in the configured region, returning a flattened slice with ID, status, node type, engine, engine version, and node count.
func (c *Client) DescribeCacheClusters(ctx context.Context) ([]CacheCluster, error) {
	result, err := c.ec.DescribeCacheClusters(ctx, &awselasticache.DescribeCacheClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cache clusters: %w", err)
	}

	clusters := make([]CacheCluster, 0, len(result.CacheClusters))
	for _, cc := range result.CacheClusters {
		clusters = append(clusters, CacheCluster{
			ID:            aws.ToString(cc.CacheClusterId),
			Status:        aws.ToString(cc.CacheClusterStatus),
			NodeType:      aws.ToString(cc.CacheNodeType),
			Engine:        aws.ToString(cc.Engine),
			EngineVersion: aws.ToString(cc.EngineVersion),
			NumCacheNodes: aws.ToInt32(cc.NumCacheNodes),
		})
	}
	return clusters, nil
}
