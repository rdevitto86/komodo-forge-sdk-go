package elasticache

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awselasticache "github.com/aws/aws-sdk-go-v2/service/elasticache"
)

type ReplicationGroup struct {
	ID            string
	Status        string
	NodeType      string
	NumNodeGroups int32
	Endpoint      string
}

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

type elastiCacheAPI interface {
	DescribeReplicationGroups(ctx context.Context, params *awselasticache.DescribeReplicationGroupsInput, optFns ...func(*awselasticache.Options)) (*awselasticache.DescribeReplicationGroupsOutput, error)
	DescribeCacheClusters(ctx context.Context, params *awselasticache.DescribeCacheClustersInput, optFns ...func(*awselasticache.Options)) (*awselasticache.DescribeCacheClustersOutput, error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type Client struct {
	ec elastiCacheAPI
}

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

func newWithAPI(api elastiCacheAPI) *Client {
	return &Client{ec: api}
}

func (c *Client) DescribeReplicationGroups(ctx context.Context) ([]ReplicationGroup, error) {
	var groups []ReplicationGroup
	in := &awselasticache.DescribeReplicationGroupsInput{}

	for {
		result, err := c.ec.DescribeReplicationGroups(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("failed to describe replication groups: %w", err)
		}

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

		if result.Marker == nil {
			break
		}
		in.Marker = result.Marker
	}

	return groups, nil
}

func (c *Client) DescribeCacheClusters(ctx context.Context) ([]CacheCluster, error) {
	var clusters []CacheCluster
	in := &awselasticache.DescribeCacheClustersInput{}

	for {
		result, err := c.ec.DescribeCacheClusters(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("failed to describe cache clusters: %w", err)
		}

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

		if result.Marker == nil {
			break
		}
		in.Marker = result.Marker
	}

	return clusters, nil
}

var _ API = (*Client)(nil)
