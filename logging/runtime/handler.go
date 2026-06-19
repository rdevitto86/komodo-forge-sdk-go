package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

type textHandler struct {
	mu       sync.Mutex
	w        io.Writer
	level    slog.Leveler
	mode     RedactMode
	preAttrs []slog.Attr
}

func newTextHandler(w io.Writer, level slog.Leveler, mode RedactMode) *textHandler {
	return &textHandler{w: w, level: level, mode: mode}
}

func (h *textHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := &textHandler{w: h.w, level: h.level, mode: h.mode}
	h2.preAttrs = make([]slog.Attr, len(h.preAttrs)+len(attrs))
	copy(h2.preAttrs, h.preAttrs)
	copy(h2.preAttrs[len(h.preAttrs):], attrs)
	return h2
}

func (h *textHandler) WithGroup(_ string) slog.Handler { return h }

func (h *textHandler) Handle(_ context.Context, rec slog.Record) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	requestID := "-"
	var attrs strings.Builder
	first := true

	collect := func(a slog.Attr) {
		if a.Key == "request_id" {
			requestID = a.Value.String()
			return
		}
		a = redactAttr(h.mode, a)
		if s := formatAttr(a); s != "" {
			if !first {
				attrs.WriteByte(' ')
			}
			attrs.WriteString(s)
			first = false
		}
	}

	for _, a := range h.preAttrs {
		collect(a)
	}
	rec.Attrs(func(a slog.Attr) bool {
		collect(a)
		return true
	})

	// Format: timestamp [LEVEL] request_id | message | attributes

	buf.WriteString(rec.Time.UTC().Format(time.RFC3339)) // timestamp
	buf.WriteByte(' ')
	buf.WriteString(levelLabel(rec.Level)) // [LEVEL]
	buf.WriteByte(' ')
	buf.WriteString(requestID) // request_id
	buf.WriteString(" | ")
	buf.WriteString(rec.Message) // message

	if attrs.Len() > 0 {
		buf.WriteString(" | ")
		buf.WriteString(attrs.String()) // attributes
	}
	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	return err
}

func levelLabel(level slog.Level) string {
	switch {
	case level >= LevelFatal:
		return "[FATAL]"
	case level >= slog.LevelError:
		return "[ERROR]"
	case level >= slog.LevelWarn:
		return "[WARN]"
	case level >= slog.LevelInfo:
		return "[INFO]"
	default:
		return "[DEBUG]"
	}
}

func formatAttr(attr slog.Attr) string {
	k := attr.Key
	v := attr.Value.Resolve()

	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		if needsQuoting(s) {
			return k + "=" + strconv.Quote(s)
		}
		return k + "=" + s
	case slog.KindInt64:
		return k + "=" + strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return k + "=" + strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return fmt.Sprintf("%s=%g", k, v.Float64())
	case slog.KindBool:
		return k + "=" + strconv.FormatBool(v.Bool())
	case slog.KindGroup:
		sub := v.Group()
		if len(sub) == 0 {
			return ""
		}
		parts := make([]string, 0, len(sub))
		for _, a := range sub {
			if s := formatAttr(slog.Attr{Key: k + "." + a.Key, Value: a.Value}); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	case slog.KindAny:
		any := v.Any()
		if err, ok := any.(error); ok {
			return k + "=" + strconv.Quote(err.Error())
		}
		b, err := json.Marshal(any)
		if err != nil {
			return k + "=<error>"
		}
		s := string(b)
		if len(s) > 200 {
			s = s[:197] + "..."
		}
		if needsQuoting(s) {
			return k + "=" + strconv.Quote(s)
		}
		return k + "=" + s
	default:
		s := v.String()
		if needsQuoting(s) {
			return k + "=" + strconv.Quote(s)
		}
		return k + "=" + s
	}
}

func needsQuoting(s string) bool { return strings.ContainsAny(s, " \t\n\r\"=") }
