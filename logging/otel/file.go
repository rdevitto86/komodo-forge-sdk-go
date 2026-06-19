package otel

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type fileRecord struct {
	Timestamp  string            `json:"timestamp"`
	Severity   string            `json:"severity"`
	Body       string            `json:"body"`
	Attributes map[string]any    `json:"attributes,omitempty"`
	TraceID    string            `json:"trace_id,omitempty"`
	SpanID     string            `json:"span_id,omitempty"`
	Resource   map[string]string `json:"resource,omitempty"`
}

type fileExporter struct {
	mu   sync.Mutex
	f    *os.File
	enc  *json.Encoder
	done bool
}

func newFileExporter(path string) (*fileExporter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &fileExporter{f: f, enc: json.NewEncoder(f)}, nil
}

func (e *fileExporter) Export(_ context.Context, records []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.done {
		return nil
	}
	for _, rec := range records {
		fr := fileRecord{
			Timestamp: rec.Timestamp().UTC().Format(time.RFC3339Nano),
			Severity:  rec.Severity().String(),
			Body:      rec.Body().AsString(),
		}

		if n := rec.AttributesLen(); n > 0 {
			fr.Attributes = make(map[string]any, n)
			rec.WalkAttributes(func(kv log.KeyValue) bool {
				fr.Attributes[kv.Key] = kvToAny(kv.Value)
				return true
			})
		}

		tid := rec.TraceID()
		if tid.IsValid() {
			fr.TraceID = tid.String()
		}
		sid := rec.SpanID()
		if sid.IsValid() {
			fr.SpanID = sid.String()
		}

		if res := rec.Resource(); res != nil {
			attrs := res.Attributes()
			if len(attrs) > 0 {
				fr.Resource = make(map[string]string, len(attrs))
				for _, a := range attrs {
					fr.Resource[string(a.Key)] = a.Value.String()
				}
			}
		}

		if err := e.enc.Encode(fr); err != nil {
			return err
		}
	}
	return nil
}

func (e *fileExporter) Shutdown(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.done {
		return nil
	}
	e.done = true
	return e.f.Close()
}

func (e *fileExporter) ForceFlush(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.done {
		return nil
	}
	return e.f.Sync()
}

func kvToAny(v log.Value) any {
	switch v.Kind() {
	case log.KindString:
		return v.AsString()
	case log.KindInt64:
		return v.AsInt64()
	case log.KindFloat64:
		return v.AsFloat64()
	case log.KindBool:
		return v.AsBool()
	case log.KindBytes:
		return v.AsBytes()
	case log.KindSlice:
		sl := v.AsSlice()
		out := make([]any, len(sl))
		for i, item := range sl {
			out[i] = kvToAny(item)
		}
		return out
	case log.KindMap:
		m := v.AsMap()
		out := make(map[string]any, len(m))
		for _, kv := range m {
			out[kv.Key] = kvToAny(kv.Value)
		}
		return out
	default:
		return nil
	}
}
