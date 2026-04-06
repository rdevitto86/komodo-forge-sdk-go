package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

// --- helpers ---

// captureLogger replaces the global slogger with a buffer-based logger for the test.
func captureLogger(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slogger
	slogger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level}))
	t.Cleanup(func() { slogger = old })
	return &buf
}

// resetInitOnce allows Init to be called again in subsequent tests.
func resetInitOnce() {
	initOnce = sync.Once{}
}

// ===================== logger.go =====================

func TestRuntimeLogger_Init_LocalEnv_Success(t *testing.T) {
	old := slogger
	resetInitOnce()
	t.Cleanup(func() {
		slogger = old
		resetInitOnce()
	})

	Init("my-service", "debug", "local", "v1.0")
	if slogger == nil {
		t.Error("expected slogger to be non-nil after Init")
	}
}

func TestRuntimeLogger_Init_ProdEnv_Success(t *testing.T) {
	old := slogger
	resetInitOnce()
	t.Cleanup(func() {
		slogger = old
		resetInitOnce()
	})

	Init("my-service", "info", "prod")
	if slogger == nil {
		t.Error("expected slogger to be non-nil after Init")
	}
}

func TestRuntimeLogger_Init_NoVersion_Success(t *testing.T) {
	old := slogger
	resetInitOnce()
	t.Cleanup(func() {
		slogger = old
		resetInitOnce()
	})

	// No version arg → defaults to "unknown"; must not panic.
	Init("svc", "warn", "dev")
}

func TestRuntimeLogger_Init_Idempotent_Success(t *testing.T) {
	old := slogger
	resetInitOnce()
	t.Cleanup(func() {
		slogger = old
		resetInitOnce()
	})

	Init("svc", "info", "staging")
	first := slogger
	Init("svc", "debug", "local") // second call must be a no-op
	if slogger != first {
		t.Error("expected Init to be idempotent (slogger should not change on second call)")
	}
}

func TestRuntimeLogger_Debug_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelDebug)
	Debug("debug message", "key", "val")
	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("expected 'debug message' in output, got: %q", buf.String())
	}
}

func TestRuntimeLogger_Info_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelInfo)
	Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Errorf("expected 'info message' in output, got: %q", buf.String())
	}
}

func TestRuntimeLogger_Warn_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelWarn)
	Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("expected 'warn message' in output, got: %q", buf.String())
	}
}

func TestRuntimeLogger_Error_WithError_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelError)
	Error("error occurred", errors.New("something failed"))
	out := buf.String()
	if !strings.Contains(out, "error occurred") {
		t.Errorf("expected 'error occurred' in output, got: %q", out)
	}
}

func TestRuntimeLogger_Error_NilError_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelError)
	Error("no-error message", nil)
	if !strings.Contains(buf.String(), "no-error message") {
		t.Errorf("expected message in output, got: %q", buf.String())
	}
}

func TestRuntimeLogger_Fatal_WithError_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelError)
	Fatal("fatal event", errors.New("fatal error"))
	out := buf.String()
	if !strings.Contains(out, "fatal event") {
		t.Errorf("expected 'fatal event' in output, got: %q", out)
	}
}

func TestRuntimeLogger_Fatal_NilError_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelError)
	Fatal("fatal no-error", nil)
	if !strings.Contains(buf.String(), "fatal no-error") {
		t.Errorf("expected message in output, got: %q", buf.String())
	}
}

func TestRuntimeLogger_SetLevel_Success(t *testing.T) {
	// SetLevel changes the shared logLevel; just verify no panic.
	SetLevel("debug")
	SetLevel("info")
	SetLevel("warn")
	SetLevel("error")
	SetLevel("unknown") // falls to LevelError
	SetLevel("")        // falls to LevelInfo
}

func TestRuntimeLogger_IsLocalEnv_Success(t *testing.T) {
	for _, env := range []string{"local", "LOCAL", "dev", "DEV", "development", "DEVELOPMENT"} {
		if !isLocalEnv(env) {
			t.Errorf("isLocalEnv(%q) = false, want true", env)
		}
	}
}

func TestRuntimeLogger_IsLocalEnv_Failure(t *testing.T) {
	for _, env := range []string{"prod", "staging", "production", ""} {
		if isLocalEnv(env) {
			t.Errorf("isLocalEnv(%q) = true, want false", env)
		}
	}
}

func TestRuntimeLogger_ParseLevel_Success(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"", slog.LevelInfo},
		{"error", slog.LevelError},
		{"unknown", slog.LevelError},
	}
	for _, tc := range tests {
		got := parseLevel(tc.input)
		if got != tc.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ===================== wrappers.go =====================

func TestLoggerWrapper_Attr_Success(t *testing.T) {
	a := Attr("custom_key", "custom_value")
	if a.Key != "custom_key" {
		t.Errorf("Key = %q, want custom_key", a.Key)
	}
}

func TestLoggerWrapper_AttrError_Success(t *testing.T) {
	a := AttrError(errors.New("test error"))
	if a.Key != "error" {
		t.Errorf("Key = %q, want error", a.Key)
	}
}

func TestLoggerWrapper_AttrRequestID_Success(t *testing.T) {
	a := AttrRequestID("req-123")
	if a.Key != "request_id" {
		t.Errorf("Key = %q, want request_id", a.Key)
	}
	if a.Value.String() != "req-123" {
		t.Errorf("Value = %q, want req-123", a.Value.String())
	}
}

func TestLoggerWrapper_AttrCorrelationID_Success(t *testing.T) {
	a := AttrCorrelationID("corr-456")
	if a.Key != "correlation_id" {
		t.Errorf("Key = %q, want correlation_id", a.Key)
	}
}

func TestLoggerWrapper_AttrUserID_Success(t *testing.T) {
	a := AttrUserID("user-789")
	if a.Key != "user_id" {
		t.Errorf("Key = %q, want user_id", a.Key)
	}
}

func TestLoggerWrapper_AttrSessionID_Success(t *testing.T) {
	a := AttrSessionID("sess-abc")
	if a.Key != "session_id" {
		t.Errorf("Key = %q, want session_id", a.Key)
	}
}

func TestLoggerWrapper_AttrDetails_Success(t *testing.T) {
	a := AttrDetails(map[string]any{"foo": "bar"})
	if a.Key != "details" {
		t.Errorf("Key = %q, want details", a.Key)
	}
}

func TestLoggerWrapper_FromContext_WithValues_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxKeys.REQUEST_ID_KEY, "req-1")
	ctx = context.WithValue(ctx, ctxKeys.CORRELATION_ID_KEY, "corr-1")
	ctx = context.WithValue(ctx, ctxKeys.USER_ID_KEY, "user-1")
	ctx = context.WithValue(ctx, ctxKeys.SESSION_ID_KEY, "sess-1")

	args := FromContext(ctx)
	if len(args) != 4 {
		t.Errorf("expected 4 args, got %d", len(args))
	}
}

func TestLoggerWrapper_FromContext_Empty_Success(t *testing.T) {
	args := FromContext(context.Background())
	if len(args) != 0 {
		t.Errorf("expected 0 args for empty context, got %d", len(args))
	}
}

func TestLoggerWrapper_FromContext_PartialValues_Success(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKeys.REQUEST_ID_KEY, "req-only")
	args := FromContext(ctx)
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d: %v", len(args), args)
	}
}

func TestLoggerWrapper_AttrRequest_WithRequest_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	a := AttrRequest(req)
	if a.Key != "request" {
		t.Errorf("Key = %q, want request", a.Key)
	}
}

func TestLoggerWrapper_AttrRequest_NilRequest_Success(t *testing.T) {
	a := AttrRequest(nil)
	if a.Key != "request" {
		t.Errorf("Key = %q, want request", a.Key)
	}
}

func TestLoggerWrapper_AttrResponse_WithResponse_Success(t *testing.T) {
	resp := &http.Response{StatusCode: 200}
	a := AttrResponse(resp)
	if a.Key != "response" {
		t.Errorf("Key = %q, want response", a.Key)
	}
}

func TestLoggerWrapper_AttrResponse_NilResponse_Success(t *testing.T) {
	a := AttrResponse(nil)
	if a.Key != "response" {
		t.Errorf("Key = %q, want response", a.Key)
	}
}

// ===================== handler.go =====================

func TestLoggerHandler_NewKomodoTextHandler_Success(t *testing.T) {
	var buf bytes.Buffer
	h := NewKomodoTextHandler(&buf, false, slog.LevelDebug)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.w != &buf {
		t.Error("expected handler.w to be the provided writer")
	}
}

func TestLoggerHandler_KomodoTextHandler_Enabled_AboveLevel_Success(t *testing.T) {
	h := NewKomodoTextHandler(&bytes.Buffer{}, false, slog.LevelWarn)
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("ERROR should be enabled when level=WARN")
	}
}

func TestLoggerHandler_KomodoTextHandler_Enabled_BelowLevel_Failure(t *testing.T) {
	h := NewKomodoTextHandler(&bytes.Buffer{}, false, slog.LevelWarn)
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("DEBUG should not be enabled when level=WARN")
	}
}

func TestLoggerHandler_KomodoTextHandler_WithAttrs_Success(t *testing.T) {
	var buf bytes.Buffer
	h := NewKomodoTextHandler(&buf, false, slog.LevelDebug)

	h2 := h.WithAttrs([]slog.Attr{slog.String("service", "test-svc")})
	if h2 == h {
		t.Error("WithAttrs should return a new handler instance")
	}
	h2typed, ok := h2.(*KomodoTextHandler)
	if !ok {
		t.Fatal("expected *KomodoTextHandler from WithAttrs")
	}
	if len(h2typed.preAttrs) != 1 {
		t.Errorf("expected 1 preAttr, got %d", len(h2typed.preAttrs))
	}
}

func TestLoggerHandler_KomodoTextHandler_WithGroup_Success(t *testing.T) {
	h := NewKomodoTextHandler(&bytes.Buffer{}, false, slog.LevelDebug)
	h2 := h.WithGroup("mygroup")
	if h2 != h {
		t.Error("WithGroup should return the same handler (no-op)")
	}
}

func TestLoggerHandler_KomodoTextHandler_Handle_BasicMessage_Success(t *testing.T) {
	var buf bytes.Buffer
	h := NewKomodoTextHandler(&buf, false, slog.LevelDebug)

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("expected message in output, got: %q", buf.String())
	}
}

func TestLoggerHandler_KomodoTextHandler_Handle_WithAttrs_Success(t *testing.T) {
	var buf bytes.Buffer
	h := NewKomodoTextHandler(&buf, false, slog.LevelDebug)

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	rec.AddAttrs(slog.String("env", "prod")) // skipped base field
	rec.AddAttrs(slog.String("request_id", "req-xyz"))
	rec.AddAttrs(slog.String("custom", "myval"))
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	out := buf.String()
	// "env" is a skipped base field and must not appear as a kv pair.
	if strings.Contains(out, "env=prod") {
		t.Errorf("skipped base field 'env' should not appear in output: %q", out)
	}
	// request_id is promoted to the requestID slot; still present in line but not as k=v.
	if !strings.Contains(out, "req-xyz") {
		t.Errorf("expected request_id value in output: %q", out)
	}
	if !strings.Contains(out, "custom=myval") {
		t.Errorf("expected 'custom=myval' in output: %q", out)
	}
}

func TestLoggerHandler_KomodoTextHandler_Handle_WithColor_Success(t *testing.T) {
	var buf bytes.Buffer
	h := NewKomodoTextHandler(&buf, true, slog.LevelDebug)

	rec := slog.NewRecord(time.Now(), slog.LevelWarn, "colored", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	// ANSI color codes should appear.
	if !strings.Contains(buf.String(), "\033[") {
		t.Errorf("expected ANSI codes in colored output, got: %q", buf.String())
	}
}

func TestLoggerHandler_KomodoTextHandler_Handle_WithColorAndParts_Success(t *testing.T) {
	var buf bytes.Buffer
	h := NewKomodoTextHandler(&buf, true, slog.LevelDebug)

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	rec.AddAttrs(slog.String("custom_key", "custom_val"))
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\033[") {
		t.Errorf("expected ANSI codes in colored output, got: %q", out)
	}
	if !strings.Contains(out, "custom_key=custom_val") {
		t.Errorf("expected custom_key in output, got: %q", out)
	}
}

func TestLoggerHandler_KomodoTextHandler_Handle_PreAttrs_Success(t *testing.T) {
	var buf bytes.Buffer
	h := NewKomodoTextHandler(&buf, false, slog.LevelDebug)
	h2 := h.WithAttrs([]slog.Attr{slog.String("pre_key", "pre_val")}).(*KomodoTextHandler)

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	if err := h2.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(buf.String(), "pre_key=pre_val") {
		t.Errorf("expected pre_key in output, got: %q", buf.String())
	}
}

func TestLoggerHandler_ColoredLevel_AllLevels_Success(t *testing.T) {
	h := NewKomodoTextHandler(&bytes.Buffer{}, false, slog.LevelDebug)
	hc := NewKomodoTextHandler(&bytes.Buffer{}, true, slog.LevelDebug)

	tests := []struct {
		level slog.Level
		label string
	}{
		{slog.LevelDebug - 1, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
		{slog.LevelError + 1, "FATAL"},
	}
	for _, tc := range tests {
		plain := h.coloredLevel(tc.level)
		if !strings.Contains(plain, tc.label) {
			t.Errorf("coloredLevel(%v) no-color = %q, missing %q", tc.level, plain, tc.label)
		}
		colored := hc.coloredLevel(tc.level)
		if !strings.Contains(colored, tc.label) {
			t.Errorf("coloredLevel(%v) color = %q, missing %q", tc.level, colored, tc.label)
		}
		if !strings.Contains(colored, "\033[") {
			t.Errorf("expected ANSI in colored output: %q", colored)
		}
	}
}

// --- formatAttr ---

func TestLoggerHandler_FormatAttr_String_NoQuote_Success(t *testing.T) {
	a := slog.String("key", "simple")
	if got := formatAttr(a); got != "key=simple" {
		t.Errorf("formatAttr string = %q, want key=simple", got)
	}
}

func TestLoggerHandler_FormatAttr_String_NeedsQuote_Success(t *testing.T) {
	a := slog.String("key", "with space")
	got := formatAttr(a)
	if !strings.HasPrefix(got, "key=") || !strings.Contains(got, "with space") {
		t.Errorf("formatAttr string with space = %q", got)
	}
}

func TestLoggerHandler_FormatAttr_Int64_Success(t *testing.T) {
	a := slog.Int64("num", 42)
	if got := formatAttr(a); got != "num=42" {
		t.Errorf("formatAttr int64 = %q, want num=42", got)
	}
}

func TestLoggerHandler_FormatAttr_Uint64_Success(t *testing.T) {
	a := slog.Uint64("u", 99)
	if got := formatAttr(a); got != "u=99" {
		t.Errorf("formatAttr uint64 = %q, want u=99", got)
	}
}

func TestLoggerHandler_FormatAttr_Float64_Success(t *testing.T) {
	a := slog.Float64("f", 3.14)
	got := formatAttr(a)
	if !strings.HasPrefix(got, "f=") {
		t.Errorf("formatAttr float64 = %q", got)
	}
}

func TestLoggerHandler_FormatAttr_Bool_Success(t *testing.T) {
	if got := formatAttr(slog.Bool("ok", true)); got != "ok=true" {
		t.Errorf("formatAttr bool = %q, want ok=true", got)
	}
}

func TestLoggerHandler_FormatAttr_Group_Success(t *testing.T) {
	a := slog.Group("req", slog.String("method", "GET"), slog.String("path", "/"))
	got := formatAttr(a)
	if !strings.Contains(got, "req.method=GET") {
		t.Errorf("formatAttr group = %q, missing req.method=GET", got)
	}
}

func TestLoggerHandler_FormatAttr_Group_Empty_Success(t *testing.T) {
	a := slog.Group("empty")
	if got := formatAttr(a); got != "" {
		t.Errorf("formatAttr empty group = %q, want empty", got)
	}
}

func TestLoggerHandler_FormatAttr_AnyError_Success(t *testing.T) {
	a := slog.Any("err", errors.New("boom"))
	got := formatAttr(a)
	if !strings.Contains(got, "err=") || !strings.Contains(got, "boom") {
		t.Errorf("formatAttr any error = %q", got)
	}
}

func TestLoggerHandler_FormatAttr_AnyJSON_Success(t *testing.T) {
	a := slog.Any("data", map[string]string{"x": "y"})
	got := formatAttr(a)
	if !strings.HasPrefix(got, "data=") {
		t.Errorf("formatAttr any json = %q", got)
	}
}

func TestLoggerHandler_FormatAttr_AnyLongJSON_Truncated_Success(t *testing.T) {
	large := make([]int, 100) // JSON marshal > 200 chars
	for i := range large {
		large[i] = i
	}
	a := slog.Any("big", large)
	got := formatAttr(a)
	if !strings.Contains(got, "...") {
		t.Errorf("expected truncation with '...', got: %q", got)
	}
}

func TestLoggerHandler_FormatAttr_AnyMarshalError_Success(t *testing.T) {
	// Channels cannot be JSON-encoded — exercises the k+"=<error>" branch.
	a := slog.Any("ch", make(chan int))
	got := formatAttr(a)
	if !strings.Contains(got, "<error>") {
		t.Errorf("expected '<error>' for non-marshallable value, got: %q", got)
	}
}

func TestLoggerHandler_FormatAttr_Duration_Success(t *testing.T) {
	a := slog.Duration("dur", time.Second)
	got := formatAttr(a)
	if !strings.HasPrefix(got, "dur=") {
		t.Errorf("formatAttr duration = %q", got)
	}
}

func TestLoggerHandler_FormatAttr_Time_Success(t *testing.T) {
	a := slog.Time("ts", time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC))
	got := formatAttr(a)
	if !strings.HasPrefix(got, "ts=") {
		t.Errorf("formatAttr time = %q", got)
	}
}

// --- needsQuoting ---

func TestLoggerHandler_NeedsQuoting_WithSpace_Success(t *testing.T) {
	if !needsQuoting("has space") {
		t.Error("expected needsQuoting=true for string with space")
	}
}

func TestLoggerHandler_NeedsQuoting_WithEquals_Success(t *testing.T) {
	if !needsQuoting("key=val") {
		t.Error("expected needsQuoting=true for string with '='")
	}
}

func TestLoggerHandler_NeedsQuoting_WithQuote_Success(t *testing.T) {
	if !needsQuoting(`say "hi"`) {
		t.Error("expected needsQuoting=true for string with '\"'")
	}
}

func TestLoggerHandler_NeedsQuoting_Plain_Failure(t *testing.T) {
	if needsQuoting("simple-value") {
		t.Error("expected needsQuoting=false for plain string")
	}
}

// ===================== redaction.go =====================

func TestLoggerRedaction_Handle_Success(t *testing.T) {
	var buf bytes.Buffer
	inner := NewKomodoTextHandler(&buf, false, slog.LevelDebug)
	rl := &RedactingLogger{Handler: inner}

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "test redact", 0)
	rec.AddAttrs(slog.String("authorization", "Bearer secret-token"))
	rec.AddAttrs(slog.String("user", "alice"))

	if err := rl.Handle(context.Background(), rec); err != nil {
		t.Fatalf("RedactingLogger.Handle: %v", err)
	}
	// Must produce output without panicking.
	if buf.Len() == 0 {
		t.Error("expected non-empty output from RedactingLogger.Handle")
	}
}

// Ensure fmt is used (used in formatAttr via fmt.Sprintf for float64).
var _ = fmt.Sprintf
