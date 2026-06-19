package otel

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// ── Helpers ──────────────────────────────────────────────────────────────

type memoryExporter struct {
	records []sdklog.Record
}

func (e *memoryExporter) Export(_ context.Context, records []sdklog.Record) error {
	for _, r := range records {
		e.records = append(e.records, r.Clone())
	}
	return nil
}

func (e *memoryExporter) Shutdown(context.Context) error    { return nil }
func (e *memoryExporter) ForceFlush(context.Context) error  { return nil }

func newTestLogger(t *testing.T, level log.Severity) (*OtelLogger, *memoryExporter) {
	t.Helper()
	exp := &memoryExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exp)),
	)
	t.Cleanup(func() { provider.Shutdown(context.Background()) })
	return &OtelLogger{
		provider: provider,
		logger:   provider.Logger("test"),
		level:    level,
	}, exp
}

// ── Unit Tests ──────────────────────────────────────────────────────────

func TestOtelLogger_New_NoOutput_Error(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error when no output configured")
	}
	if !strings.Contains(err.Error(), "at least one output") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOtelLogger_New_WithFile_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	l, err := New(Config{
		FilePath:       path,
		ServiceName:    "test-svc",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer l.Shutdown(context.Background())

	l.Info(context.Background(), "hello from file", String("user", "alice"))
	l.ForceFlush(context.Background())

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, "hello from file") {
		t.Errorf("expected log body in file, got %q", raw)
	}
	if !strings.Contains(raw, `"user"`) {
		t.Errorf("expected attribute key in file, got %q", raw)
	}
	if !strings.Contains(raw, "test-svc") {
		t.Errorf("expected resource service.name in file, got %q", raw)
	}
}

func TestOtelLogger_Trace_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityTrace)
	l.Trace(context.Background(), "trace msg", String("k", "v"))

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	rec := exp.records[0]
	if rec.Severity() != log.SeverityTrace {
		t.Errorf("expected TRACE severity, got %v", rec.Severity())
	}
	if rec.Body().AsString() != "trace msg" {
		t.Errorf("expected body 'trace msg', got %q", rec.Body().AsString())
	}
}

func TestOtelLogger_Debug_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityDebug)
	l.Debug(context.Background(), "debug msg")

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	if exp.records[0].Severity() != log.SeverityDebug {
		t.Errorf("expected DEBUG severity, got %v", exp.records[0].Severity())
	}
}

func TestOtelLogger_Info_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityInfo)
	l.Info(context.Background(), "info msg")

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	if exp.records[0].Severity() != log.SeverityInfo {
		t.Errorf("expected INFO severity, got %v", exp.records[0].Severity())
	}
}

func TestOtelLogger_Warn_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityWarn)
	l.Warn(context.Background(), "warn msg")

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	if exp.records[0].Severity() != log.SeverityWarn {
		t.Errorf("expected WARN severity, got %v", exp.records[0].Severity())
	}
}

func TestOtelLogger_Error_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityError)
	l.Error(context.Background(), "error msg")

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	if exp.records[0].Severity() != log.SeverityError {
		t.Errorf("expected ERROR severity, got %v", exp.records[0].Severity())
	}
}

func TestOtelLogger_Fatal_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityFatal)
	l.Fatal(context.Background(), "fatal msg")

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	if exp.records[0].Severity() != log.SeverityFatal {
		t.Errorf("expected FATAL severity, got %v", exp.records[0].Severity())
	}
}

func TestOtelLogger_LevelFiltering_BelowThreshold(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityWarn)
	l.Debug(context.Background(), "should be dropped")
	l.Info(context.Background(), "also dropped")
	l.Warn(context.Background(), "kept")
	l.Error(context.Background(), "also kept")

	if len(exp.records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(exp.records))
	}
	if exp.records[0].Body().AsString() != "kept" {
		t.Errorf("expected 'kept', got %q", exp.records[0].Body().AsString())
	}
	if exp.records[1].Body().AsString() != "also kept" {
		t.Errorf("expected 'also kept', got %q", exp.records[1].Body().AsString())
	}
}

func TestOtelLogger_Enabled_Success(t *testing.T) {
	l, _ := newTestLogger(t, log.SeverityWarn)
	if l.Enabled(context.Background(), log.SeverityDebug) {
		t.Error("DEBUG should not be enabled at WARN level")
	}
	if !l.Enabled(context.Background(), log.SeverityWarn) {
		t.Error("WARN should be enabled at WARN level")
	}
	if !l.Enabled(context.Background(), log.SeverityError) {
		t.Error("ERROR should be enabled at WARN level")
	}
}

func TestOtelLogger_Publish_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityTrace)
	var rec log.Record
	rec.SetTimestamp(time.Now())
	rec.SetSeverity(log.SeverityInfo)
	rec.SetBody(log.StringValue("raw record"))
	l.Publish(context.Background(), rec)

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	if exp.records[0].Body().AsString() != "raw record" {
		t.Errorf("expected 'raw record', got %q", exp.records[0].Body().AsString())
	}
}

func TestOtelLogger_WithAttributes_Success(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityTrace)
	child := l.WithAttributes(String("env", "test"), Int("version", 1))

	child.Info(context.Background(), "child msg", Bool("extra", true))

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}

	rec := exp.records[0]
	found := map[string]bool{}
	rec.WalkAttributes(func(kv log.KeyValue) bool {
		found[kv.Key] = true
		return true
	})
	for _, key := range []string{"env", "version", "extra"} {
		if !found[key] {
			t.Errorf("expected attribute %q", key)
		}
	}
}

func TestOtelLogger_WithAttributes_DoesNotMutateParent(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityTrace)
	_ = l.WithAttributes(String("child_attr", "x"))

	l.Info(context.Background(), "parent msg")

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	rec := exp.records[0]
	rec.WalkAttributes(func(kv log.KeyValue) bool {
		if kv.Key == "child_attr" {
			t.Error("parent should not have child's attribute")
		}
		return true
	})
}

func TestOtelLogger_Attributes_WithCallAttributes(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityTrace)
	l.Info(context.Background(), "attrs test",
		String("s", "val"),
		Int("i", 42),
		Int64("i64", 100),
		Float64("f", 3.14),
		Bool("b", true),
		Bytes("raw", []byte{0x01, 0x02}),
	)

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	count := 0
	exp.records[0].WalkAttributes(func(kv log.KeyValue) bool {
		count++
		return true
	})
	if count != 6 {
		t.Errorf("expected 6 attributes, got %d", count)
	}
}

func TestOtelLogger_Shutdown_Success(t *testing.T) {
	l, _ := newTestLogger(t, log.SeverityInfo)
	if err := l.Shutdown(context.Background()); err != nil {
		t.Errorf("expected nil error from Shutdown, got %v", err)
	}
}

func TestOtelLogger_ForceFlush_Success(t *testing.T) {
	l, _ := newTestLogger(t, log.SeverityInfo)
	if err := l.ForceFlush(context.Background()); err != nil {
		t.Errorf("expected nil error from ForceFlush, got %v", err)
	}
}

func TestOtelLogger_DefaultLevel_Info(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	l, err := New(Config{FilePath: path})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer l.Shutdown(context.Background())

	if l.level != log.SeverityInfo {
		t.Errorf("expected default level INFO, got %v", l.level)
	}
}

func TestOtelLogger_Timestamp_Set(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityTrace)
	before := time.Now()
	l.Info(context.Background(), "ts test")
	after := time.Now()

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	ts := exp.records[0].Timestamp()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not between %v and %v", ts, before, after)
	}
}

func TestOtelLogger_SeverityText_Set(t *testing.T) {
	l, exp := newTestLogger(t, log.SeverityTrace)
	l.Warn(context.Background(), "sev text")

	if len(exp.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exp.records))
	}
	if exp.records[0].SeverityText() != "WARN" {
		t.Errorf("expected severity text 'WARN', got %q", exp.records[0].SeverityText())
	}
}

// ── File Exporter Tests ─────────────────────────────────────────────────

func TestFileExporter_Export_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.log")
	fe, err := newFileExporter(path)
	if err != nil {
		t.Fatalf("failed to create file exporter: %v", err)
	}
	defer fe.Shutdown(context.Background())

	var rec sdklog.Record
	rec.SetTimestamp(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))
	rec.SetSeverity(log.SeverityInfo)
	rec.SetBody(log.StringValue("test entry"))

	if err := fe.Export(context.Background(), []sdklog.Record{rec}); err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var fr fileRecord
	if err := json.Unmarshal(data, &fr); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if fr.Body != "test entry" {
		t.Errorf("expected body 'test entry', got %q", fr.Body)
	}
	if fr.Severity != "INFO" {
		t.Errorf("expected severity 'INFO', got %q", fr.Severity)
	}
	if fr.Timestamp != "2025-01-15T10:30:00Z" {
		t.Errorf("expected timestamp, got %q", fr.Timestamp)
	}
}

func TestFileExporter_MultipleRecords_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multi.log")
	fe, err := newFileExporter(path)
	if err != nil {
		t.Fatalf("failed to create file exporter: %v", err)
	}
	defer fe.Shutdown(context.Background())

	records := make([]sdklog.Record, 3)
	for i := range records {
		records[i].SetTimestamp(time.Now())
		records[i].SetSeverity(log.SeverityInfo)
		records[i].SetBody(log.StringValue("line"))
	}

	if err := fe.Export(context.Background(), records); err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestFileExporter_ShutdownPreventsExport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shutdown.log")
	fe, err := newFileExporter(path)
	if err != nil {
		t.Fatalf("failed to create file exporter: %v", err)
	}

	fe.Shutdown(context.Background())

	var rec sdklog.Record
	rec.SetBody(log.StringValue("after shutdown"))
	if err := fe.Export(context.Background(), []sdklog.Record{rec}); err != nil {
		t.Fatalf("expected nil error after shutdown, got %v", err)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "after shutdown") {
		t.Error("expected no records after shutdown")
	}
}

func TestFileExporter_ForceFlush_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flush.log")
	fe, err := newFileExporter(path)
	if err != nil {
		t.Fatalf("failed to create file exporter: %v", err)
	}
	defer fe.Shutdown(context.Background())

	if err := fe.ForceFlush(context.Background()); err != nil {
		t.Errorf("expected nil error from ForceFlush, got %v", err)
	}
}

func TestFileExporter_InvalidPath_Error(t *testing.T) {
	_, err := newFileExporter("/nonexistent/dir/file.log")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestFileExporter_DoubleShutdown_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "double.log")
	fe, err := newFileExporter(path)
	if err != nil {
		t.Fatalf("failed to create file exporter: %v", err)
	}
	if err := fe.Shutdown(context.Background()); err != nil {
		t.Errorf("first shutdown error: %v", err)
	}
	if err := fe.Shutdown(context.Background()); err != nil {
		t.Errorf("second shutdown should be idempotent, got: %v", err)
	}
}

// ── Resource Tests ──────────────────────────────────────────────────────

func TestBuildResource_Defaults(t *testing.T) {
	res, err := buildResource(Config{})
	if err != nil {
		t.Fatalf("failed to build resource: %v", err)
	}
	attrs := res.Attributes()
	found := false
	for _, a := range attrs {
		if string(a.Key) == "service.name" && a.Value.AsString() == "unknown_service" {
			found = true
		}
	}
	if !found {
		t.Error("expected default service.name 'unknown_service'")
	}
}

func TestBuildResource_ConfigValues(t *testing.T) {
	res, err := buildResource(Config{
		ServiceName:    "my-svc",
		ServiceVersion: "1.2.3",
		Environment:    "staging",
	})
	if err != nil {
		t.Fatalf("failed to build resource: %v", err)
	}
	attrs := res.Attributes()
	want := map[string]string{
		"service.name":            "my-svc",
		"service.version":         "1.2.3",
		"deployment.environment":  "staging",
	}
	got := make(map[string]string, len(attrs))
	for _, a := range attrs {
		got[string(a.Key)] = a.Value.AsString()
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("resource %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestBuildResource_EnvVarFallback(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "env-svc")
	t.Setenv("OTEL_SERVICE_VERSION", "0.1.0")
	t.Setenv("OTEL_DEPLOYMENT_ENVIRONMENT", "production")

	res, err := buildResource(Config{})
	if err != nil {
		t.Fatalf("failed to build resource: %v", err)
	}
	attrs := res.Attributes()
	got := make(map[string]string, len(attrs))
	for _, a := range attrs {
		got[string(a.Key)] = a.Value.AsString()
	}
	if got["service.name"] != "env-svc" {
		t.Errorf("expected env-svc, got %q", got["service.name"])
	}
	if got["service.version"] != "0.1.0" {
		t.Errorf("expected 0.1.0, got %q", got["service.version"])
	}
	if got["deployment.environment"] != "production" {
		t.Errorf("expected production, got %q", got["deployment.environment"])
	}
}

func TestBuildResource_ConfigOverridesEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "env-svc")
	res, err := buildResource(Config{ServiceName: "config-svc"})
	if err != nil {
		t.Fatalf("failed to build resource: %v", err)
	}
	for _, a := range res.Attributes() {
		if string(a.Key) == "service.name" && a.Value.AsString() != "config-svc" {
			t.Errorf("config should override env, got %q", a.Value.AsString())
		}
	}
}
