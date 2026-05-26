// LocalStack community does not support Bedrock. Tests are component-only via
// SDK interface mocking. Integration coverage requires a real AWS account with
// the relevant model access granted.
package bedrock

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

// fakeRuntimeAPI implements bedrockRuntimeAPI for component tests.
type fakeRuntimeAPI struct {
	invokeModelFunc func(ctx context.Context, in *bedrockruntime.InvokeModelInput, opts ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
	converseFunc    func(ctx context.Context, in *bedrockruntime.ConverseInput, opts ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

func (f *fakeRuntimeAPI) InvokeModel(ctx context.Context, in *bedrockruntime.InvokeModelInput, opts ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	if f.invokeModelFunc != nil {
		return f.invokeModelFunc(ctx, in, opts...)
	}
	return nil, errors.New("InvokeModel not configured on fake")
}

func (f *fakeRuntimeAPI) Converse(ctx context.Context, in *bedrockruntime.ConverseInput, opts ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	if f.converseFunc != nil {
		return f.converseFunc(ctx, in, opts...)
	}
	return nil, errors.New("Converse not configured on fake")
}

// ── New tests ─────────────────────────────────────────────────────────────────

func TestNew_EmptyRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for empty region, got nil")
	}
	if err == nil || err.Error() != "missing region" {
		t.Errorf("got %v, want \"missing region\"", err)
	}
}

// ── InvokeJSON tests ──────────────────────────────────────────────────────────

func TestInvokeJSON_PassThrough(t *testing.T) {
	rawBody := []byte(`{"custom":"payload","value":42}`)
	wantResp := []byte(`{"result":"ok"}`)

	var capturedBody []byte
	fake := &fakeRuntimeAPI{
		invokeModelFunc: func(_ context.Context, in *bedrockruntime.InvokeModelInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			capturedBody = in.Body
			return &bedrockruntime.InvokeModelOutput{Body: wantResp}, nil
		},
	}
	c := newWithAPI(fake)

	got, err := c.InvokeJSON(context.Background(), ModelMistralLarge, rawBody)
	if err != nil {
		t.Fatalf("InvokeJSON returned unexpected error: %v", err)
	}
	if string(capturedBody) != string(rawBody) {
		t.Errorf("InvokeModel received body %q, want %q", capturedBody, rawBody)
	}
	if string(got) != string(wantResp) {
		t.Errorf("InvokeJSON returned %q, want %q", got, wantResp)
	}
}

func TestInvokeJSON_RejectsUnknownModel(t *testing.T) {
	called := false
	fake := &fakeRuntimeAPI{
		invokeModelFunc: func(_ context.Context, _ *bedrockruntime.InvokeModelInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			called = true
			return nil, nil
		},
	}
	c := newWithAPI(fake)

	_, err := c.InvokeJSON(context.Background(), Model("not-real"), []byte(`{}`))
	if err == nil {
		t.Fatal("InvokeJSON expected error for unknown model, got nil")
	}
	if !errors.Is(err, ErrUnknownModel) {
		t.Errorf("InvokeJSON error = %v, want to wrap ErrUnknownModel", err)
	}
	if called {
		t.Error("InvokeJSON called the SDK despite an invalid model")
	}
}

// ── Invoke tests ──────────────────────────────────────────────────────────────

func TestInvoke_RejectsUnknownModel(t *testing.T) {
	called := false
	fake := &fakeRuntimeAPI{
		invokeModelFunc: func(_ context.Context, _ *bedrockruntime.InvokeModelInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			called = true
			return nil, nil
		},
	}
	c := newWithAPI(fake)

	_, err := c.Invoke(context.Background(), Model("not-real"), "hello")
	if err == nil {
		t.Fatal("Invoke expected error for unknown model, got nil")
	}
	if !errors.Is(err, ErrUnknownModel) {
		t.Errorf("Invoke error = %v, want to wrap ErrUnknownModel", err)
	}
	if called {
		t.Error("Invoke called the SDK despite an invalid model")
	}
}

func TestInvoke_AnthropicBuildsBody(t *testing.T) {
	const prompt = "What is the meaning of life?"

	var capturedInput *bedrockruntime.InvokeModelInput
	// Build a valid Anthropic-format response so Invoke can parse it back.
	respPayload, _ := json.Marshal(map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": "42"}},
	})
	fake := &fakeRuntimeAPI{
		invokeModelFunc: func(_ context.Context, in *bedrockruntime.InvokeModelInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			capturedInput = in
			return &bedrockruntime.InvokeModelOutput{Body: respPayload}, nil
		},
	}
	c := newWithAPI(fake)

	text, err := c.Invoke(context.Background(), ModelClaudeSonnet4_6, prompt)
	if err != nil {
		t.Fatalf("Invoke returned unexpected error: %v", err)
	}
	if text != "42" {
		t.Errorf("Invoke returned %q, want %q", text, "42")
	}

	// Verify the body sent to InvokeModel is well-formed Anthropic JSON.
	var body anthropicRequestBody
	if err := json.Unmarshal(capturedInput.Body, &body); err != nil {
		t.Fatalf("captured body is not valid JSON: %v", err)
	}
	if body.AnthropicVersion == "" {
		t.Error("anthropic_version is empty in request body")
	}
	if body.MaxTokens <= 0 {
		t.Error("max_tokens must be positive in request body")
	}
	if len(body.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(body.Messages))
	}
	if body.Messages[0].Role != "user" {
		t.Errorf("message role = %q, want %q", body.Messages[0].Role, "user")
	}
	if body.Messages[0].Content != prompt {
		t.Errorf("message content = %q, want %q", body.Messages[0].Content, prompt)
	}
}

func TestInvoke_NoTextContentReturnsError(t *testing.T) {
	// Response contains only a non-text block; Invoke must error rather than return "".
	respPayload, _ := json.Marshal(map[string]interface{}{
		"content": []map[string]string{{"type": "image", "text": ""}},
	})
	fake := &fakeRuntimeAPI{
		invokeModelFunc: func(_ context.Context, _ *bedrockruntime.InvokeModelInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			return &bedrockruntime.InvokeModelOutput{Body: respPayload}, nil
		},
	}
	c := newWithAPI(fake)

	_, err := c.Invoke(context.Background(), ModelClaudeSonnet4_6, "hello")
	if err == nil {
		t.Fatal("Invoke expected error when response has no text content, got nil")
	}
	const wantSubstr = "no text content"
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("Invoke error = %q, want it to contain %q", err.Error(), wantSubstr)
	}
}

func TestInvoke_NonAnthropicReturnsError(t *testing.T) {
	fake := &fakeRuntimeAPI{}
	c := newWithAPI(fake)

	_, err := c.Invoke(context.Background(), ModelMistralLarge, "hi")
	if err == nil {
		t.Fatal("Invoke expected error for non-Anthropic model, got nil")
	}
	// Must not wrap ErrUnknownModel — the model is valid, just unsupported by Invoke.
	if errors.Is(err, ErrUnknownModel) {
		t.Errorf("Invoke error should not be ErrUnknownModel for a valid non-Anthropic model: %v", err)
	}
}

// ── Converse tests ────────────────────────────────────────────────────────────

func TestConverse_HappyPath(t *testing.T) {
	const wantText = "Hello! I am doing well."
	inputTokens := int32(10)
	outputTokens := int32(8)

	fake := &fakeRuntimeAPI{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return &bedrockruntime.ConverseOutput{
				Output: &types.ConverseOutputMemberMessage{
					Value: types.Message{
						Role: types.ConversationRoleAssistant,
						Content: []types.ContentBlock{
							&types.ContentBlockMemberText{Value: wantText},
						},
					},
				},
				StopReason: types.StopReasonEndTurn,
				Usage: &types.TokenUsage{
					InputTokens:  &inputTokens,
					OutputTokens: &outputTokens,
				},
			}, nil
		},
	}
	c := newWithAPI(fake)

	out, err := c.Converse(context.Background(), ConverseInput{
		Model: ModelClaudeHaiku4_5,
		Messages: []Message{
			{Role: "user", Content: "How are you?"},
		},
		MaxTokens: 512,
	})
	if err != nil {
		t.Fatalf("Converse returned unexpected error: %v", err)
	}
	if out.Text != wantText {
		t.Errorf("Converse text = %q, want %q", out.Text, wantText)
	}
	if out.StopReason != string(types.StopReasonEndTurn) {
		t.Errorf("Converse stop reason = %q, want %q", out.StopReason, types.StopReasonEndTurn)
	}
	if out.InputTokens != inputTokens {
		t.Errorf("InputTokens = %d, want %d", out.InputTokens, inputTokens)
	}
	if out.OutputTokens != outputTokens {
		t.Errorf("OutputTokens = %d, want %d", out.OutputTokens, outputTokens)
	}
}

func TestConverse_RejectsUnknownModel(t *testing.T) {
	called := false
	fake := &fakeRuntimeAPI{
		converseFunc: func(_ context.Context, _ *bedrockruntime.ConverseInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			called = true
			return nil, nil
		},
	}
	c := newWithAPI(fake)

	_, err := c.Converse(context.Background(), ConverseInput{
		Model:    Model("not-real"),
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("Converse expected error for unknown model, got nil")
	}
	if !errors.Is(err, ErrUnknownModel) {
		t.Errorf("Converse error = %v, want to wrap ErrUnknownModel", err)
	}
	if called {
		t.Error("Converse called the SDK despite an invalid model")
	}
}
