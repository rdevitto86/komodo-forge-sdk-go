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

const (
	ansiReset   = "\033[0m"
	ansiGray    = "\033[90m"
	ansiCyan    = "\033[36m"
	ansiYellow  = "\033[33m"
	ansiRed     = "\033[31m"
	ansiBoldRed = "\033[1;31m"
)

var skippedBaseFields = map[string]bool{
	"service": true,
	"env":     true,
	"version": true,
}

var (
	plainFatal = "[FATAL]"
	plainError = "[ERROR]"
	plainWarn  = "[WARN]"
	plainInfo  = "[INFO]"
	plainDebug = "[DEBUG]"
	colorFatal = ansiBoldRed + "[FATAL]" + ansiReset
	colorError = ansiRed + "[ERROR]" + ansiReset
	colorWarn  = ansiYellow + "[WARN]" + ansiReset
	colorInfo  = ansiCyan + "[INFO]" + ansiReset
	colorDebug = ansiGray + "[DEBUG]" + ansiReset
)

var bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

type KomodoTextHandler struct {
	mu       sync.Mutex
	w        io.Writer
	color    bool
	level    slog.Leveler
	preAttrs []slog.Attr // attrs set via .With() (e.g., service, env, version)
}

func NewKomodoTextHandler(w io.Writer, color bool, level slog.Leveler) *KomodoTextHandler {
	return &KomodoTextHandler{w: w, color: color, level: level}
}

func (h *KomodoTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *KomodoTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := &KomodoTextHandler{
		w:     h.w,
		color: h.color,
		level: h.level,
	}
	h2.preAttrs = make([]slog.Attr, len(h.preAttrs)+len(attrs))
	copy(h2.preAttrs, h.preAttrs)
	copy(h2.preAttrs[len(h.preAttrs):], attrs)
	return h2
}

func (h *KomodoTextHandler) WithGroup(name string) slog.Handler { return h }

func (h *KomodoTextHandler) Handle(_ context.Context, rec slog.Record) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	requestID := "-"
	var attrs strings.Builder
	first := true

	collectAttr := func(attr slog.Attr) {
		if skippedBaseFields[attr.Key] {
			return
		}
		if attr.Key == "request_id" {
			requestID = attr.Value.String()
			return
		}
		if s := formatAttr(attr); s != "" {
			if !first {
				attrs.WriteByte(' ')
			}
			attrs.WriteString(s)
			first = false
		}
	}

	for _, a := range h.preAttrs {
		collectAttr(a)
	}
	rec.Attrs(func(a slog.Attr) bool {
		collectAttr(a)
		return true
	})

	// timestamp
	if h.color {
		buf.WriteString(ansiGray)
	}
	buf.WriteString(rec.Time.UTC().Format(time.RFC3339))
	if h.color {
		buf.WriteString(ansiReset)
	}
	buf.WriteByte(' ')

	// [LEVEL]
	buf.WriteString(h.coloredLevel(rec.Level))
	buf.WriteByte(' ')

	// requestId
	if h.color {
		buf.WriteString(ansiGray)
	}
	buf.WriteString(requestID)
	if h.color {
		buf.WriteString(ansiReset)
	}
	buf.WriteString(" | ")

	// message
	buf.WriteString(rec.Message)

	// details as logfmt
	if attrs.Len() > 0 {
		buf.WriteString(" | ")
		if h.color {
			buf.WriteString(ansiGray)
		}
		buf.WriteString(attrs.String())
		if h.color {
			buf.WriteString(ansiReset)
		}
	}

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *KomodoTextHandler) coloredLevel(level slog.Level) string {
	switch {
	case level > slog.LevelError:
		if h.color {
			return colorFatal
		}
		return plainFatal
	case level >= slog.LevelError:
		if h.color {
			return colorError
		}
		return plainError
	case level >= slog.LevelWarn:
		if h.color {
			return colorWarn
		}
		return plainWarn
	case level >= slog.LevelInfo:
		if h.color {
			return colorInfo
		}
		return plainInfo
	default:
		if h.color {
			return colorDebug
		}
		return plainDebug
	}
}

// Renders a slog.Attr as a logfmt key=value pair; complex values are JSON-encoded and truncated at 200 chars.
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
