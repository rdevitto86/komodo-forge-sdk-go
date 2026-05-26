package ses

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// Represents a file to attach to an outgoing email.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Carries all parameters for a SendEmail call.
type SendEmailInput struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	TextBody    string
	HTMLBody    string
	ReplyTo     []string
	Attachments []Attachment
}

// sesAPI is the interface seam over the AWS SESv2 SDK client; allows test injection without a real endpoint.
type sesAPI interface {
	SendEmail(ctx context.Context, in *sesv2.SendEmailInput, opts ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// Defines the SES operations exposed by this package.
type API interface {
	SendEmail(ctx context.Context, input SendEmailInput) (messageID string, err error)
}

// Holds connection parameters for the SES client.
type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string // optional; set to LocalStack URL in non-prod environments
}

// Wraps the AWS SESv2 SDK client.
type Client struct {
	ses sesAPI
}

// Creates and returns a new SES Client. Accepts a context for AWS config loading. Returns an error if Region is empty.
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

	var opts []func(*sesv2.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *sesv2.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{ses: sesv2.NewFromConfig(cfg, opts...)}, nil
}

// newWithAPI constructs a Client with an injected sesAPI — used only in tests.
func newWithAPI(api sesAPI) *Client {
	return &Client{ses: api}
}

// Sends an outgoing email via SESv2. Assembles raw multipart/mixed MIME when attachments are present. Returns the SES message ID.
func (c *Client) SendEmail(ctx context.Context, input SendEmailInput) (string, error) {
	if input.From == "" {
		return "", fmt.Errorf("missing from address")
	}
	if len(input.To) == 0 {
		return "", fmt.Errorf("missing To address")
	}

	var in *sesv2.SendEmailInput

	if len(input.Attachments) > 0 {
		raw, err := buildRawMessage(input)
		if err != nil {
			return "", fmt.Errorf("failed to build raw MIME message: %w", err)
		}
		in = &sesv2.SendEmailInput{
			FromEmailAddress: aws.String(input.From),
			Destination: &types.Destination{
				ToAddresses:  input.To,
				CcAddresses:  input.Cc,
				BccAddresses: input.Bcc,
			},
			Content: &types.EmailContent{
				Raw: &types.RawMessage{Data: raw},
			},
		}
		if len(input.ReplyTo) > 0 {
			in.ReplyToAddresses = input.ReplyTo
		}
	} else {
		dest := &types.Destination{
			ToAddresses:  input.To,
			CcAddresses:  input.Cc,
			BccAddresses: input.Bcc,
		}

		body := &types.Body{}
		if input.HTMLBody != "" {
			body.Html = &types.Content{Data: aws.String(input.HTMLBody)}
		}
		if input.TextBody != "" {
			body.Text = &types.Content{Data: aws.String(input.TextBody)}
		}

		in = &sesv2.SendEmailInput{
			FromEmailAddress: aws.String(input.From),
			Destination:      dest,
			Content: &types.EmailContent{
				Simple: &types.Message{
					Subject: &types.Content{Data: aws.String(input.Subject)},
					Body:    body,
				},
			},
		}

		if len(input.ReplyTo) > 0 {
			in.ReplyToAddresses = input.ReplyTo
		}
	}

	result, err := c.ses.SendEmail(ctx, in)
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}
	return aws.ToString(result.MessageId), nil
}

// Helper that constructs a multipart/mixed RFC 2822 MIME message including all body parts and base64-encoded attachments.
func buildRawMessage(input SendEmailInput) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// RFC 2822 envelope headers must appear before the MIME boundary.
	fmt.Fprintf(&buf, "From: %s\r\n", input.From)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(input.To, ", "))
	if len(input.Cc) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(input.Cc, ", "))
	}
	if len(input.ReplyTo) > 0 {
		fmt.Fprintf(&buf, "Reply-To: %s\r\n", strings.Join(input.ReplyTo, ", "))
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", input.Subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", w.Boundary())

	if input.TextBody != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/plain; charset=UTF-8")
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		pw, err := w.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("failed to create text part: %w", err)
		}
		fmt.Fprint(pw, input.TextBody)
	}

	if input.HTMLBody != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/html; charset=UTF-8")
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		pw, err := w.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("failed to create html part: %w", err)
		}
		fmt.Fprint(pw, input.HTMLBody)
	}

	for _, att := range input.Attachments {
		h := make(textproto.MIMEHeader)
		ct := att.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)
		h.Set("Content-Transfer-Encoding", "base64")
		h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", att.Filename))

		pw, err := w.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("failed to create attachment part: %w", err)
		}
		enc := base64.NewEncoder(base64.StdEncoding, pw)
		if _, err := enc.Write(att.Data); err != nil {
			return nil, fmt.Errorf("failed to encode attachment data: %w", err)
		}
		if err := enc.Close(); err != nil {
			return nil, fmt.Errorf("failed to finalise attachment encoding: %w", err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close MIME writer: %w", err)
	}

	return buf.Bytes(), nil
}
