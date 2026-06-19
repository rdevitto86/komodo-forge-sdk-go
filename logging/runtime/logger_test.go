package logger

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func captureLogger(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slogger.Load()
	slogger.Store(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { slogger.Store(old) })
	return &buf
}

func captureInit(t *testing.T, cfg Config) *bytes.Buffer {
	t.Helper()
	oldLogger := slogger.Load()
	oldLevel := logLevel.Level()
	oldDst := stdoutDst
	buf := new(bytes.Buffer)
	stdoutDst = buf
	if err := Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		Close()
		slogger.Store(oldLogger)
		logLevel.Set(oldLevel)
		stdoutDst = oldDst
	})
	return buf
}

func captureExit(t *testing.T) *int {
	t.Helper()
	code := -1
	old := osExit
	osExit = func(c int) { code = c }
	t.Cleanup(func() { osExit = old })
	return &code
}

func TestRuntimeLogger_Debug_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelDebug)
	Debug("debug message", "key", "val")
	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("expected 'debug message', got %q", buf.String())
	}
}

func TestRuntimeLogger_Info_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelInfo)
	Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Errorf("expected 'info message', got %q", buf.String())
	}
}

func TestRuntimeLogger_Warn_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelWarn)
	Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("expected 'warn message', got %q", buf.String())
	}
}

func TestRuntimeLogger_Error_WithError_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelError)
	Error("error occurred", errors.New("something failed"))
	if !strings.Contains(buf.String(), "error occurred") {
		t.Errorf("expected 'error occurred', got %q", buf.String())
	}
}

func TestRuntimeLogger_Error_NilError_Success(t *testing.T) {
	buf := captureLogger(t, slog.LevelError)
	Error("no-error message", nil)
	if !strings.Contains(buf.String(), "no-error message") {
		t.Errorf("expected message, got %q", buf.String())
	}
}

func TestRuntimeLogger_SetLevel_Success(t *testing.T) {
	old := logLevel.Level()
	t.Cleanup(func() { logLevel.Set(old) })
	SetLevel("debug")
	if logLevel.Level() != slog.LevelDebug {
		t.Errorf("expected debug level, got %v", logLevel.Level())
	}
	SetLevel("error")
	if logLevel.Level() != slog.LevelError {
		t.Errorf("expected error level, got %v", logLevel.Level())
	}
}

func TestRuntimeLogger_Enabled_Success(t *testing.T) {
	old := logLevel.Level()
	t.Cleanup(func() { logLevel.Set(old) })

	logLevel.Set(slog.LevelInfo)
	tests := []struct {
		level string
		want  bool
	}{
		{"debug", false},
		{"info", true},
		{"warn", true},
		{"error", true},
	}
	for _, tc := range tests {
		if got := Enabled(tc.level); got != tc.want {
			t.Errorf("Enabled(%q) = %v, want %v", tc.level, got, tc.want)
		}
	}
}

func TestRuntimeLogger_DebugEnabled_Success(t *testing.T) {
	old := logLevel.Level()
	t.Cleanup(func() { logLevel.Set(old) })

	logLevel.Set(slog.LevelInfo)
	if DebugEnabled() {
		t.Error("DebugEnabled() = true, want false when level=info")
	}
	logLevel.Set(slog.LevelDebug)
	if !DebugEnabled() {
		t.Error("DebugEnabled() = false, want true when level=debug")
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
		{"warn", slog.LevelWarn},
		{"", slog.LevelInfo},
		{"error", slog.LevelError},
		{"unknown", slog.LevelError},
	}
	for _, tc := range tests {
		if got := parseLevel(tc.input); got != tc.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestInit_FormatText_Success(t *testing.T) {
	buf := captureInit(t, Config{Level: "debug", Format: FormatText})
	Info("probe", slog.String("k", "v"))
	Sync()
	out := buf.String()
	if !strings.Contains(out, "[INFO]") || !strings.Contains(out, "probe") || !strings.Contains(out, "k=v") {
		t.Errorf("unexpected text output: %q", out)
	}
}

func TestInit_FormatJSON_DefaultRedactsStrict(t *testing.T) {
	buf := captureInit(t, Config{Level: "debug"})
	Info("probe", slog.String("authorization", "Bearer supersecret-xyz"), slog.String("user", "alice"))
	Sync()
	out := buf.String()
	if strings.Contains(out, "supersecret-xyz") {
		t.Errorf("expected default Strict redaction, got %q", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected user attr, got %q", out)
	}
	if !strings.Contains(out, `"msg":"probe"`) {
		t.Errorf("expected json output, got %q", out)
	}
}

func TestInit_RedactOff_PassesThrough(t *testing.T) {
	buf := captureInit(t, Config{Level: "debug", Redact: RedactOff})
	Info("probe", slog.String("authorization", "Bearer supersecret-xyz"))
	Sync()
	if !strings.Contains(buf.String(), "supersecret-xyz") {
		t.Errorf("expected passthrough, got %q", buf.String())
	}
}

func TestInit_TextWithSinks_Errors(t *testing.T) {
	err := Init(Config{Format: FormatText, Sinks: []Sink{{URL: "http://example"}}})
	if err == nil {
		t.Error("expected error for FormatText with remote sinks")
	}
}

func TestFatal_TextLabelAndExit(t *testing.T) {
	buf := captureInit(t, Config{Level: "debug", Format: FormatText})
	code := captureExit(t)
	Fatal("boom", errors.New("bad"))
	out := buf.String()
	if !strings.Contains(out, "[FATAL]") {
		t.Errorf("expected FATAL label, got %q", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("expected message, got %q", out)
	}
	if *code != 1 {
		t.Errorf("expected exit code 1, got %d", *code)
	}
}

func TestRedaction_RedactAttr_Modes(t *testing.T) {
	if got := redactAttr(RedactOff, slog.String("authorization", "secret-token-value")); got.Value.String() != "secret-token-value" {
		t.Errorf("Off should not redact, got %q", got.Value.String())
	}
	if got := redactAttr(RedactKeysOnly, slog.String("authorization", "x")); got.Value.String() != "[REDACTED]" {
		t.Errorf("KeysOnly should redact sensitive key, got %q", got.Value.String())
	}
	if got := redactAttr(RedactKeysOnly, slog.String("note", "email me at a@b.com")); !strings.Contains(got.Value.String(), "a@b.com") {
		t.Errorf("KeysOnly should leave value PII, got %q", got.Value.String())
	}
	if got := redactAttr(RedactStrict, slog.String("note", "email me at a@b.com")); strings.Contains(got.Value.String(), "a@b.com") {
		t.Errorf("Strict should redact value PII, got %q", got.Value.String())
	}
}

func TestRedaction_RedactAttr_DeepGroupAndMap(t *testing.T) {
	g := redactAttr(RedactStrict, slog.Group("acct", slog.String("ssn", "123-45-6789"), slog.String("name", "alice")))
	if out := formatAttr(g); strings.Contains(out, "123-45-6789") {
		t.Errorf("nested ssn should be redacted, got %q", out)
	}
	m := redactAttr(RedactStrict, slog.Any("blob", map[string]any{"password": "hunter2", "ok": "fine"}))
	if out := formatAttr(m); strings.Contains(out, "hunter2") {
		t.Errorf("map password should be redacted, got %q", out)
	} else if !strings.Contains(out, "fine") {
		t.Errorf("non-sensitive map value should remain, got %q", out)
	}
}

func TestRedaction_JSONReplaceAttr(t *testing.T) {
	f := jsonReplaceAttr(RedactStrict)
	if got := f(nil, slog.Any(slog.LevelKey, LevelFatal)); got.Value.String() != "FATAL" {
		t.Errorf("expected FATAL level mapping, got %q", got.Value.String())
	}
	if got := f(nil, slog.String(slog.MessageKey, "keep me")); got.Value.String() != "keep me" {
		t.Errorf("message should be untouched, got %q", got.Value.String())
	}
	if got := f(nil, slog.String("password", "x")); got.Value.String() != "[REDACTED]" {
		t.Errorf("sensitive key should be redacted, got %q", got.Value.String())
	}

	off := jsonReplaceAttr(RedactOff)
	if got := off(nil, slog.String("authorization", "secret")); got.Value.String() != "secret" {
		t.Errorf("Off should not redact, got %q", got.Value.String())
	}
}

func TestHandler_Text_FormatAndRedact(t *testing.T) {
	var buf bytes.Buffer
	h := newTextHandler(&buf, slog.LevelDebug, RedactStrict)
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)
	rec.AddAttrs(
		slog.String("request_id", "req-1"),
		slog.String("user", "alice"),
		slog.String("authorization", "Bearer abc.def.ghi"),
	)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected level label, got %q", out)
	}
	if !strings.Contains(out, "req-1 | hello") {
		t.Errorf("expected 'req-1 | hello' layout, got %q", out)
	}
	if !strings.Contains(out, "user=alice") {
		t.Errorf("expected user attr, got %q", out)
	}
	if strings.Contains(out, "abc.def.ghi") {
		t.Errorf("expected authorization redacted, got %q", out)
	}
}

func TestHandler_Text_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := newTextHandler(&buf, slog.LevelDebug, RedactOff)
	h2 := h.WithAttrs([]slog.Attr{slog.String("pre", "val")}).(*textHandler)
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	if err := h2.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(buf.String(), "pre=val") {
		t.Errorf("expected pre attr, got %q", buf.String())
	}
}

func TestHandler_LevelLabel(t *testing.T) {
	cases := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "[DEBUG]"},
		{slog.LevelInfo, "[INFO]"},
		{slog.LevelWarn, "[WARN]"},
		{slog.LevelError, "[ERROR]"},
		{LevelFatal, "[FATAL]"},
	}
	for _, c := range cases {
		if got := levelLabel(c.level); got != c.want {
			t.Errorf("levelLabel(%v) = %q, want %q", c.level, got, c.want)
		}
	}
}

func TestWriter_AsyncWriter_FlushAndClose(t *testing.T) {
	var buf bytes.Buffer
	w := newAsyncWriter(&buf, 16)
	w.Write([]byte("line1\n"))
	w.Flush()
	if !strings.Contains(buf.String(), "line1") {
		t.Errorf("expected flushed data, got %q", buf.String())
	}
	w.Write([]byte("line2\n"))
	w.Close()
	if !strings.Contains(buf.String(), "line2") {
		t.Errorf("expected drained data after Close, got %q", buf.String())
	}
}

func TestWriter_Fanout_WritesAll(t *testing.T) {
	var a, b bytes.Buffer
	f := &fanout{writers: []io.Writer{&a, &b}}
	f.Write([]byte("x"))
	if a.String() != "x" || b.String() != "x" {
		t.Errorf("fanout did not write to all: a=%q b=%q", a.String(), b.String())
	}
}

func TestWriter_HTTPSink_PostsPayload(t *testing.T) {
	var mu sync.Mutex
	var got []byte
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, b...)
		gotKey = r.Header.Get("Api-Key")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := newHTTPSink(Sink{URL: srv.URL, Headers: map[string]string{"Api-Key": "k123"}})
	s.Write([]byte(`{"msg":"hi"}` + "\n"))
	s.Close()

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(string(got), `"msg":"hi"`) {
		t.Errorf("sink did not deliver payload, got %q", got)
	}
	if gotKey != "k123" {
		t.Errorf("expected auth header forwarded, got %q", gotKey)
	}
}

func TestInit_WithSink_FansOut(t *testing.T) {
	var mu sync.Mutex
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, b...)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	oldLogger := slogger.Load()
	oldLevel := logLevel.Level()
	oldDst := stdoutDst
	var local bytes.Buffer
	stdoutDst = &local
	if err := Init(Config{Level: "debug", Format: FormatJSON, Sinks: []Sink{{URL: srv.URL}}}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		Close()
		slogger.Store(oldLogger)
		logLevel.Set(oldLevel)
		stdoutDst = oldDst
	})

	Info("fanout probe")
	Sync()
	Close()

	if !strings.Contains(local.String(), "fanout probe") {
		t.Errorf("expected local stdout to receive log, got %q", local.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(string(got), "fanout probe") {
		t.Errorf("expected remote sink to receive log, got %q", got)
	}
}

func TestHandler_FormatAttr(t *testing.T) {
	if got := formatAttr(slog.String("key", "simple")); got != "key=simple" {
		t.Errorf("string = %q", got)
	}
	if got := formatAttr(slog.Int64("num", 42)); got != "num=42" {
		t.Errorf("int64 = %q", got)
	}
	if got := formatAttr(slog.Uint64("u", 99)); got != "u=99" {
		t.Errorf("uint64 = %q", got)
	}
	if got := formatAttr(slog.Bool("ok", true)); got != "ok=true" {
		t.Errorf("bool = %q", got)
	}
	if got := formatAttr(slog.Float64("f", 3.14)); !strings.HasPrefix(got, "f=") {
		t.Errorf("float64 = %q", got)
	}
	if got := formatAttr(slog.Group("req", slog.String("method", "GET"))); !strings.Contains(got, "req.method=GET") {
		t.Errorf("group = %q", got)
	}
	if got := formatAttr(slog.Group("empty")); got != "" {
		t.Errorf("empty group = %q", got)
	}
	if got := formatAttr(slog.Any("err", errors.New("boom"))); !strings.Contains(got, "boom") {
		t.Errorf("any error = %q", got)
	}
	if got := formatAttr(slog.Any("data", map[string]string{"x": "y"})); !strings.HasPrefix(got, "data=") {
		t.Errorf("any json = %q", got)
	}
	large := make([]int, 100)
	if got := formatAttr(slog.Any("big", large)); !strings.Contains(got, "...") {
		t.Errorf("expected truncation, got %q", got)
	}
	if got := formatAttr(slog.Any("ch", make(chan int))); !strings.Contains(got, "<error>") {
		t.Errorf("expected <error>, got %q", got)
	}
	if got := formatAttr(slog.Duration("dur", time.Second)); !strings.HasPrefix(got, "dur=") {
		t.Errorf("duration = %q", got)
	}
}

func TestHandler_NeedsQuoting(t *testing.T) {
	if !needsQuoting("has space") {
		t.Error("expected quoting for space")
	}
	if !needsQuoting("key=val") {
		t.Error("expected quoting for '='")
	}
	if needsQuoting("simple-value") {
		t.Error("expected no quoting for plain string")
	}
}

func TestWrapper_Attrs(t *testing.T) {
	if a := Attr("custom_key", "v"); a.Key != "custom_key" {
		t.Errorf("Attr key = %q", a.Key)
	}
	if a := AttrError(errors.New("x")); a.Key != "error" {
		t.Errorf("AttrError key = %q", a.Key)
	}
	if a := AttrRequestID("req-123"); a.Key != "request_id" || a.Value.String() != "req-123" {
		t.Errorf("AttrRequestID = %v", a)
	}
	if a := AttrCorrelationID("c"); a.Key != "correlation_id" {
		t.Errorf("AttrCorrelationID key = %q", a.Key)
	}
	if a := AttrUserID("u"); a.Key != "user_id" {
		t.Errorf("AttrUserID key = %q", a.Key)
	}
	if a := AttrSessionID("s"); a.Key != "session_id" {
		t.Errorf("AttrSessionID key = %q", a.Key)
	}
	if a := AttrDetails(map[string]any{"foo": "bar"}); a.Key != "details" {
		t.Errorf("AttrDetails key = %q", a.Key)
	}
}

func TestWrapper_FromContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxKeys.REQUEST_ID_KEY, "req-1")
	ctx = context.WithValue(ctx, ctxKeys.CORRELATION_ID_KEY, "corr-1")
	ctx = context.WithValue(ctx, ctxKeys.USER_ID_KEY, "user-1")
	ctx = context.WithValue(ctx, ctxKeys.SESSION_ID_KEY, "sess-1")
	if args := FromContext(ctx); len(args) != 4 {
		t.Errorf("expected 4 args, got %d", len(args))
	}
	if args := FromContext(context.Background()); len(args) != 0 {
		t.Errorf("expected 0 args, got %d", len(args))
	}
}

func TestWrapper_RequestResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	if a := AttrRequest(req); a.Key != "request" {
		t.Errorf("AttrRequest key = %q", a.Key)
	}
	if a := AttrRequest(nil); a.Key != "request" {
		t.Errorf("AttrRequest(nil) key = %q", a.Key)
	}
	if a := AttrResponse(&http.Response{StatusCode: 200}); a.Key != "response" {
		t.Errorf("AttrResponse key = %q", a.Key)
	}
	if a := AttrResponse(nil); a.Key != "response" {
		t.Errorf("AttrResponse(nil) key = %q", a.Key)
	}
}
