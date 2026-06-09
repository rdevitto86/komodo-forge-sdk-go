package connect

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/connect"
)

type API interface {
	StartOutboundVoiceContact(ctx context.Context, input OutboundVoiceContactInput) (string, error)
	GetContactAttributes(ctx context.Context, instanceID, contactID string) (map[string]string, error)
	UpdateContactAttributes(ctx context.Context, instanceID, contactID string, attrs map[string]string) error
	ListContactFlows(ctx context.Context, instanceID string) ([]ContactFlow, error)
}

// Interface seam over the AWS SDK client; allows test injection without a real endpoint.
type connectAPI interface {
	StartOutboundVoiceContact(ctx context.Context, in *connect.StartOutboundVoiceContactInput, opts ...func(*connect.Options)) (*connect.StartOutboundVoiceContactOutput, error)
	GetContactAttributes(ctx context.Context, in *connect.GetContactAttributesInput, opts ...func(*connect.Options)) (*connect.GetContactAttributesOutput, error)
	UpdateContactAttributes(ctx context.Context, in *connect.UpdateContactAttributesInput, opts ...func(*connect.Options)) (*connect.UpdateContactAttributesOutput, error)
	ListContactFlows(ctx context.Context, in *connect.ListContactFlowsInput, opts ...func(*connect.Options)) (*connect.ListContactFlowsOutput, error)
}

type OutboundVoiceContactInput struct {
	InstanceID       string
	ContactFlowID    string
	DestinationPhone string
	SourcePhone      string
	Attributes       map[string]string
}

// Flattens the Connect ContactFlowSummary type; only ID, ARN, Name, and Type are populated.
type ContactFlow struct {
	ID   string
	ARN  string
	Name string
	Type string
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type Client struct {
	api connectAPI
}

// Constructs a Connect Client from the supplied Config. Region is required; AccessKey+SecretKey enable static credentials, Endpoint alone injects test credentials for LocalStack.
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

	var opts []func(*connect.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *connect.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{api: connect.NewFromConfig(cfg, opts...)}, nil
}

// Constructs a Client backed by a supplied fake; used in tests only.
func newWithAPI(api connectAPI) *Client {
	return &Client{api: api}
}

// Places an outbound call via the specified contact flow and returns the assigned contact ID.
func (c *Client) StartOutboundVoiceContact(ctx context.Context, input OutboundVoiceContactInput) (string, error) {
	if input.InstanceID == "" {
		return "", fmt.Errorf("missing InstanceID")
	}
	if input.ContactFlowID == "" {
		return "", fmt.Errorf("missing ContactFlowID")
	}
	if input.DestinationPhone == "" {
		return "", fmt.Errorf("missing DestinationPhone")
	}

	in := &connect.StartOutboundVoiceContactInput{
		InstanceId:             aws.String(input.InstanceID),
		ContactFlowId:          aws.String(input.ContactFlowID),
		DestinationPhoneNumber: aws.String(input.DestinationPhone),
	}
	if input.SourcePhone != "" {
		in.SourcePhoneNumber = aws.String(input.SourcePhone)
	}
	if len(input.Attributes) > 0 {
		in.Attributes = input.Attributes
	}

	out, err := c.api.StartOutboundVoiceContact(ctx, in)
	if err != nil {
		return "", fmt.Errorf("failed to start outbound voice contact: %w", err)
	}
	return aws.ToString(out.ContactId), nil
}

// Retrieves all contact attributes for the given contact; returns an empty map when none are set.
func (c *Client) GetContactAttributes(ctx context.Context, instanceID, contactID string) (map[string]string, error) {
	if instanceID == "" {
		return nil, fmt.Errorf("missing instanceID")
	}
	if contactID == "" {
		return nil, fmt.Errorf("missing contactID")
	}

	out, err := c.api.GetContactAttributes(ctx, &connect.GetContactAttributesInput{
		InstanceId:       aws.String(instanceID),
		InitialContactId: aws.String(contactID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get contact attributes: %w", err)
	}
	if out.Attributes == nil {
		return map[string]string{}, nil
	}
	return out.Attributes, nil
}

// Sets or updates the given attributes on the contact; attrs must be non-empty.
func (c *Client) UpdateContactAttributes(ctx context.Context, instanceID, contactID string, attrs map[string]string) error {
	if instanceID == "" {
		return fmt.Errorf("missing instanceID")
	}
	if contactID == "" {
		return fmt.Errorf("missing contactID")
	}
	if len(attrs) == 0 {
		return fmt.Errorf("missing attrs")
	}

	_, err := c.api.UpdateContactAttributes(ctx, &connect.UpdateContactAttributesInput{
		InstanceId:       aws.String(instanceID),
		InitialContactId: aws.String(contactID),
		Attributes:       attrs,
	})
	if err != nil {
		return fmt.Errorf("failed to update contact attributes: %w", err)
	}
	return nil
}

// Returns all contact flows for the given Connect instance, paginating transparently.
func (c *Client) ListContactFlows(ctx context.Context, instanceID string) ([]ContactFlow, error) {
	if instanceID == "" {
		return nil, fmt.Errorf("missing instanceID")
	}

	var flows []ContactFlow
	var nextToken *string

	for {
		out, err := c.api.ListContactFlows(ctx, &connect.ListContactFlowsInput{
			InstanceId: aws.String(instanceID),
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list contact flows: %w", err)
		}

		for _, s := range out.ContactFlowSummaryList {
			flows = append(flows, ContactFlow{
				ID:   aws.ToString(s.Id),
				ARN:  aws.ToString(s.Arn),
				Name: aws.ToString(s.Name),
				Type: string(s.ContactFlowType),
			})
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return flows, nil
}

var _ API = (*Client)(nil)
