package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// Interface seam over the AWS SDK client; allows test injection without a real endpoint.
type bedrockRuntimeAPI interface {
	InvokeModel(ctx context.Context, in *bedrockruntime.InvokeModelInput, opts ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
	Converse(ctx context.Context, in *bedrockruntime.ConverseInput, opts ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

type ConverseInput struct {
	Model       Model
	Messages    []Message
	System      string
	MaxTokens   int32
	Temperature *float32
}

type ConverseOutput struct {
	Text         string
	StopReason   string
	InputTokens  int32
	OutputTokens int32
}

type API interface {
	Invoke(ctx context.Context, model Model, prompt string) (string, error)
	InvokeJSON(ctx context.Context, model Model, body []byte) ([]byte, error)
	Converse(ctx context.Context, input ConverseInput) (*ConverseOutput, error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
}

type Client struct {
	api bedrockRuntimeAPI
}

// Creates a Bedrock Client from the provided Config. Region is required; AccessKey+SecretKey enable static credentials, Endpoint alone injects test credentials for LocalStack.
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
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	var opts []func(*bedrockruntime.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *bedrockruntime.Options) {
			o.BaseEndpoint = aws.String(ep)
		})
	}

	return &Client{api: bedrockruntime.NewFromConfig(cfg, opts...)}, nil
}

// Constructs a Client backed by a supplied fake; used in tests only.
func newWithAPI(api bedrockRuntimeAPI) *Client {
	return &Client{api: api}
}

type anthropicRequestBody struct {
	AnthropicVersion string             `json:"anthropic_version"`
	MaxTokens        int                `json:"max_tokens"`
	Messages         []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Sends a prompt to the named Anthropic model using a preconfigured request envelope. Returns an error for non-Anthropic model families.
func (c *Client) Invoke(ctx context.Context, model Model, prompt string) (string, error) {
	if !model.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrUnknownModel, model)
	}
	if !strings.HasPrefix(string(model), "anthropic.") {
		return "", fmt.Errorf("convenience Invoke not implemented for model family of %s", model)
	}

	reqBody := anthropicRequestBody{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        1024,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal invoke request body: %w", err)
	}

	respBody, err := c.InvokeJSON(ctx, model, body)
	if err != nil {
		return "", err
	}

	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal invoke response: %w", err)
	}
	for _, block := range resp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("invoke response contained no text content")
}

// Sends a raw JSON body to the specified model via InvokeModel and returns the raw response bytes. The caller is responsible for marshalling and unmarshalling according to the model's native format.
func (c *Client) InvokeJSON(ctx context.Context, model Model, body []byte) ([]byte, error) {
	if !model.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrUnknownModel, model)
	}

	out, err := c.api.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(model.String()),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke model: %w", err)
	}
	return out.Body, nil
}

// Sends a multi-turn conversation to the specified model using the Bedrock Converse API and returns the model's text response and token usage.
func (c *Client) Converse(ctx context.Context, input ConverseInput) (*ConverseOutput, error) {
	if !input.Model.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrUnknownModel, input.Model)
	}

	msgs := make([]types.Message, 0, len(input.Messages))
	for _, m := range input.Messages {
		role := types.ConversationRoleUser
		if m.Role == "assistant" {
			role = types.ConversationRoleAssistant
		}
		msgs = append(msgs, types.Message{
			Role: role,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: m.Content},
			},
		})
	}

	in := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(input.Model.String()),
		Messages: msgs,
	}

	if input.System != "" {
		in.System = []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: input.System},
		}
	}

	if input.MaxTokens > 0 || input.Temperature != nil {
		inferCfg := &types.InferenceConfiguration{}
		if input.MaxTokens > 0 {
			inferCfg.MaxTokens = aws.Int32(input.MaxTokens)
		}
		if input.Temperature != nil {
			t := *input.Temperature
			inferCfg.Temperature = &t
		}
		in.InferenceConfig = inferCfg
	}

	out, err := c.api.Converse(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to converse: %w", err)
	}

	result := &ConverseOutput{
		StopReason: string(out.StopReason),
	}

	if out.Usage != nil {
		if out.Usage.InputTokens != nil {
			result.InputTokens = *out.Usage.InputTokens
		}
		if out.Usage.OutputTokens != nil {
			result.OutputTokens = *out.Usage.OutputTokens
		}
	}

	if msg, ok := out.Output.(*types.ConverseOutputMemberMessage); ok {
		for _, block := range msg.Value.Content {
			if tb, ok := block.(*types.ContentBlockMemberText); ok {
				result.Text = tb.Value
				break
			}
		}
	}

	return result, nil
}
