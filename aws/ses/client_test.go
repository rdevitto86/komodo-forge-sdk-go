package ses

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

func localstackConfig() Config {
	ep := os.Getenv("LOCALSTACK_ENDPOINT")
	if ep == "" {
		ep = "http://localhost:4566"
	}
	return Config{Region: "us-east-1", AccessKey: "test", SecretKey: "test", Endpoint: ep}
}

func skipIfNoLocalStack(t *testing.T) {
	t.Helper()
	if os.Getenv("LOCALSTACK_ENDPOINT") != "" {
		return
	}
	conn, err := net.DialTimeout("tcp", "localhost:4566", 5*time.Second)
	if err != nil {
		t.Skip("localstack not reachable; skipping component test")
	}
	conn.Close()
}

type fakeSESAPI struct {
	captured *sesv2.SendEmailInput
	msgID    string
	err      error
}

func (f *fakeSESAPI) SendEmail(_ context.Context, in *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	f.captured = in
	if f.err != nil {
		return nil, f.err
	}
	id := f.msgID
	if id == "" {
		id = "fake-message-id"
	}
	return &sesv2.SendEmailOutput{MessageId: aws.String(id)}, nil
}

// ── Unit Tests ───────────────────────────────────────────────────────────────

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
	if err == nil || err.Error() != "missing region" {
		t.Errorf("got %v, want \"missing region\"", err)
	}
}

func TestNew_ValidConfig(t *testing.T) {
	c, err := New(context.Background(), Config{Region: "us-east-1", AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestSendEmail_MissingFrom(t *testing.T) {
	c := newWithAPI(&fakeSESAPI{})
	_, err := c.SendEmail(context.Background(), SendEmailInput{
		To:      []string{"to@example.com"},
		Subject: "test",
	})
	if err == nil {
		t.Fatal("expected error for missing from address, got nil")
	}
	if err.Error() != "missing from address" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestSendEmail_MissingTo(t *testing.T) {
	c := newWithAPI(&fakeSESAPI{})
	_, err := c.SendEmail(context.Background(), SendEmailInput{
		From:    "from@example.com",
		Subject: "test",
	})
	if err == nil {
		t.Fatal("expected error for missing to address, got nil")
	}
	if err.Error() != "missing To address" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestSendEmail_BCC_PropagatedWithAttachments(t *testing.T) {
	fake := &fakeSESAPI{}
	c := newWithAPI(fake)

	input := SendEmailInput{
		From:     "sender@example.com",
		To:       []string{"to@example.com"},
		Cc:       []string{"cc@example.com"},
		Bcc:      []string{"bcc@example.com"},
		Subject:  "BCC test",
		TextBody: "Hello",
		ReplyTo:  []string{"reply@example.com"},
		Attachments: []Attachment{
			{Filename: "f.txt", ContentType: "text/plain", Data: []byte("data")},
		},
	}

	msgID, err := c.SendEmail(context.Background(), input)
	if err != nil {
		t.Fatalf("SendEmail returned unexpected error: %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty message ID")
	}

	in := fake.captured
	if in == nil {
		t.Fatal("sesAPI.SendEmail was not called")
	}
	if in.Destination == nil {
		t.Fatal("Destination is nil on raw send input")
	}
	if len(in.Destination.BccAddresses) != 1 || in.Destination.BccAddresses[0] != "bcc@example.com" {
		t.Errorf("BccAddresses = %v, want [bcc@example.com]", in.Destination.BccAddresses)
	}
	if len(in.Destination.ToAddresses) != 1 || in.Destination.ToAddresses[0] != "to@example.com" {
		t.Errorf("ToAddresses = %v, want [to@example.com]", in.Destination.ToAddresses)
	}
	if len(in.Destination.CcAddresses) != 1 || in.Destination.CcAddresses[0] != "cc@example.com" {
		t.Errorf("CcAddresses = %v, want [cc@example.com]", in.Destination.CcAddresses)
	}
	if aws.ToString(in.FromEmailAddress) != "sender@example.com" {
		t.Errorf("FromEmailAddress = %q, want sender@example.com", aws.ToString(in.FromEmailAddress))
	}
	if len(in.ReplyToAddresses) != 1 || in.ReplyToAddresses[0] != "reply@example.com" {
		t.Errorf("ReplyToAddresses = %v, want [reply@example.com]", in.ReplyToAddresses)
	}
}

func TestBuildRawMessage_WithAttachment(t *testing.T) {
	input := SendEmailInput{
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Subject:  "Test attachment",
		TextBody: "Hello world",
		Attachments: []Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Data:        []byte("file content"),
			},
		},
	}

	raw, err := buildRawMessage(input)
	if err != nil {
		t.Fatalf("buildRawMessage returned error: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected non-empty raw message")
	}

	body := string(raw)
	for _, want := range []string{
		"From: sender@example.com",
		"To: recipient@example.com",
		"Subject: Test attachment",
		"Content-Type: text/plain",
		"filename=\"test.txt\"",
		"base64",
	} {
		if !contains(body, want) {
			t.Errorf("raw message missing expected content %q", want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSendEmail_RejectsHeaderInjection(t *testing.T) {
	c := newWithAPI(&fakeSESAPI{})
	_, err := c.SendEmail(context.Background(), SendEmailInput{
		From:     "from@example.com",
		To:       []string{"to@example.com"},
		Subject:  "Subject\r\nBcc: victim@example.com",
		TextBody: "hi",
	})
	if err == nil {
		t.Fatal("expected error for CRLF in subject, got nil")
	}
}

func TestBuildRawMessage_QuotedPrintableEncodesEquals(t *testing.T) {
	raw, err := buildRawMessage(SendEmailInput{
		From:     "from@example.com",
		To:       []string{"to@example.com"},
		Subject:  "test",
		TextBody: "a=b url?x=1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(raw)
	if !contains(body, "quoted-printable") {
		t.Fatal("expected quoted-printable transfer encoding")
	}
	if !contains(body, "=3D") {
		t.Fatal("expected '=' to be quoted-printable encoded as =3D")
	}
}

func TestBuildRawMessage_MultipartAlternativeWhenBothBodies(t *testing.T) {
	raw, err := buildRawMessage(SendEmailInput{
		From:     "from@example.com",
		To:       []string{"to@example.com"},
		Subject:  "test",
		TextBody: "plain",
		HTMLBody: "<p>html</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(string(raw), "multipart/alternative") {
		t.Fatal("expected multipart/alternative when both text and html bodies set")
	}
}

// ── Integration Tests ────────────────────────────────────────────────────────

func TestLocalStack_SendEmail_Simple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	cfg := localstackConfig()
	c, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name  string
		input SendEmailInput
	}{
		{
			name: "text only",
			input: SendEmailInput{
				From:     "sender@example.com",
				To:       []string{"recipient@example.com"},
				Subject:  "Text only test",
				TextBody: "Hello from LocalStack",
			},
		},
		{
			name: "html and text",
			input: SendEmailInput{
				From:     "sender@example.com",
				To:       []string{"recipient@example.com"},
				Cc:       []string{"cc@example.com"},
				Subject:  "HTML and text test",
				TextBody: "Plain text fallback",
				HTMLBody: "<h1>Hello</h1>",
				ReplyTo:  []string{"replyto@example.com"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			msgID, err := c.SendEmail(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("SendEmail returned error: %v", err)
			}
			if msgID == "" {
				t.Fatal("expected non-empty message ID")
			}
		})
	}
}

func TestLocalStack_SendEmail_WithAttachment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component test in short mode")
	}
	skipIfNoLocalStack(t)

	cfg := localstackConfig()
	c, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	input := SendEmailInput{
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Subject:  "Attachment test",
		TextBody: "See attached",
		Attachments: []Attachment{
			{
				Filename:    "hello.txt",
				ContentType: "text/plain",
				Data:        []byte("hello attachment content"),
			},
		},
	}

	msgID, err := c.SendEmail(context.Background(), input)
	if err != nil {
		t.Fatalf("SendEmail with attachment returned error: %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty message ID")
	}

	ep := os.Getenv("LOCALSTACK_ENDPOINT")
	if ep == "" {
		ep = "http://localhost:4566"
	}
	sesMessages := fetchLocalStackSESMessages(t, ep)
	if len(sesMessages) == 0 {
		t.Log("LocalStack SES message inspection endpoint returned no messages (may require Pro)")
		return
	}
	t.Logf("LocalStack SES has %d stored message(s)", len(sesMessages))
}

func fetchLocalStackSESMessages(t *testing.T, endpoint string) []map[string]any {
	t.Helper()
	url := fmt.Sprintf("%s/_aws/ses", endpoint)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Logf("failed to fetch LocalStack SES messages: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Logf("LocalStack SES endpoint returned status %d", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("failed to read LocalStack SES response body: %v", err)
		return nil
	}

	var result struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Logf("failed to parse LocalStack SES response: %v", err)
		return nil
	}
	return result.Messages
}
