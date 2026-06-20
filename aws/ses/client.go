package ses

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

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

type sesAPI interface {
	SendEmail(ctx context.Context, in *sesv2.SendEmailInput, opts ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

type API interface {
	SendEmail(ctx context.Context, input SendEmailInput) (messageID string, err error)
}

type Config struct {
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string // optional
}

type Client struct {
	ses sesAPI
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
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	var opts []func(*sesv2.Options)
	if config.Endpoint != "" {
		ep := config.Endpoint
		opts = append(opts, func(o *sesv2.Options) { o.BaseEndpoint = aws.String(ep) })
	}

	return &Client{ses: sesv2.NewFromConfig(cfg, opts...)}, nil
}

func newWithAPI(api sesAPI) *Client {
	return &Client{ses: api}
}

func (c *Client) SendEmail(ctx context.Context, input SendEmailInput) (string, error) {
	if input.From == "" {
		return "", fmt.Errorf("missing from address")
	}
	if len(input.To) == 0 {
		return "", fmt.Errorf("missing To address")
	}
	if err := validateAddressHeaders(input); err != nil {
		return "", err
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

func buildRawMessage(input SendEmailInput) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fmt.Fprintf(&buf, "From: %s\r\n", input.From)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(input.To, ", "))
	if len(input.Cc) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(input.Cc, ", "))
	}
	if len(input.ReplyTo) > 0 {
		fmt.Fprintf(&buf, "Reply-To: %s\r\n", strings.Join(input.ReplyTo, ", "))
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", input.Subject))
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", w.Boundary())

	if err := writeBodyParts(w, input); err != nil {
		return nil, err
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

func writeBodyParts(w *multipart.Writer, input SendEmailInput) error {
	hasText := input.TextBody != ""
	hasHTML := input.HTMLBody != ""

	if hasText && hasHTML {
		var altBuf bytes.Buffer
		alt := multipart.NewWriter(&altBuf)
		if err := writeQPPart(alt, "text/plain; charset=UTF-8", input.TextBody); err != nil {
			return err
		}
		if err := writeQPPart(alt, "text/html; charset=UTF-8", input.HTMLBody); err != nil {
			return err
		}
		if err := alt.Close(); err != nil {
			return fmt.Errorf("failed to close alternative writer: %w", err)
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "multipart/alternative; boundary=\""+alt.Boundary()+"\"")
		pw, err := w.CreatePart(h)
		if err != nil {
			return fmt.Errorf("failed to create alternative part: %w", err)
		}
		if _, err := pw.Write(altBuf.Bytes()); err != nil {
			return fmt.Errorf("failed to write alternative part: %w", err)
		}
		return nil
	}

	if hasText {
		return writeQPPart(w, "text/plain; charset=UTF-8", input.TextBody)
	}
	if hasHTML {
		return writeQPPart(w, "text/html; charset=UTF-8", input.HTMLBody)
	}
	return nil
}

func writeQPPart(w *multipart.Writer, contentType, body string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", contentType)
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("failed to create body part: %w", err)
	}
	qp := quotedprintable.NewWriter(pw)
	if _, err := qp.Write([]byte(body)); err != nil {
		return fmt.Errorf("failed to encode body part: %w", err)
	}
	if err := qp.Close(); err != nil {
		return fmt.Errorf("failed to finalise body encoding: %w", err)
	}
	return nil
}

func validateAddressHeaders(input SendEmailInput) error {
	fields := []string{input.From, input.Subject}
	fields = append(fields, input.To...)
	fields = append(fields, input.Cc...)
	fields = append(fields, input.Bcc...)
	fields = append(fields, input.ReplyTo...)
	for _, f := range fields {
		if strings.ContainsAny(f, "\r\n") {
			return fmt.Errorf("failed to build email: header value contains CR or LF")
		}
	}
	return nil
}
