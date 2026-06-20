package contactlens

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/connectcontactlens"
	clstypes "github.com/aws/aws-sdk-go-v2/service/connectcontactlens/types"
)

type API interface {
	ListRealtimeContactAnalysisSegments(ctx context.Context, instanceID, contactID string) ([]Segment, error)
}

type contactLensAPI interface {
	ListRealtimeContactAnalysisSegments(ctx context.Context, in *connectcontactlens.ListRealtimeContactAnalysisSegmentsInput, opts ...func(*connectcontactlens.Options)) (*connectcontactlens.ListRealtimeContactAnalysisSegmentsOutput, error)
}

type Segment struct {
	Type              string
	Content           string
	BeginOffsetMillis int64
	EndOffsetMillis   int64
	ParticipantID     string
	Sentiment         string
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type Client struct {
	api contactLensAPI
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
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var opts []func(*connectcontactlens.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *connectcontactlens.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{api: connectcontactlens.NewFromConfig(cfg, opts...)}, nil
}

func newWithAPI(api contactLensAPI) *Client {
	return &Client{api: api}
}

func (c *Client) ListRealtimeContactAnalysisSegments(ctx context.Context, instanceID, contactID string) ([]Segment, error) {
	if instanceID == "" {
		return nil, fmt.Errorf("missing instanceID")
	}
	if contactID == "" {
		return nil, fmt.Errorf("missing contactID")
	}

	var segments []Segment
	var nextToken *string

	for {
		out, err := c.api.ListRealtimeContactAnalysisSegments(ctx, &connectcontactlens.ListRealtimeContactAnalysisSegmentsInput{
			InstanceId: aws.String(instanceID),
			ContactId:  aws.String(contactID),
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list realtime contact analysis segments: %w", err)
		}

		for _, raw := range out.Segments {
			seg := mapSegment(raw)
			segments = append(segments, seg)
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return segments, nil
}

func mapSegment(raw clstypes.RealtimeContactAnalysisSegment) Segment {
	if raw.Transcript == nil {
		return Segment{Type: "UNKNOWN"}
	}
	t := raw.Transcript
	var begin, end int64
	if t.BeginOffsetMillis != nil {
		begin = int64(*t.BeginOffsetMillis)
	}
	if t.EndOffsetMillis != nil {
		end = int64(*t.EndOffsetMillis)
	}
	return Segment{
		Type:              "TRANSCRIPT",
		Content:           aws.ToString(t.Content),
		BeginOffsetMillis: begin,
		EndOffsetMillis:   end,
		ParticipantID:     aws.ToString(t.ParticipantId),
		Sentiment:         string(t.Sentiment),
	}
}

var _ API = (*Client)(nil)
