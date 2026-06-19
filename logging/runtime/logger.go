package logger

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
)

type Format int

const (
	FormatJSON Format = iota
	FormatText
)

type RedactMode int

const (
	RedactStrict RedactMode = iota
	RedactKeysOnly
	RedactOff
)

const LevelFatal = slog.LevelError + 4

type Sink struct {
	URL     string
	Headers map[string]string
}

type Config struct {
	Level  string
	Format Format
	Redact RedactMode
	Sinks  []Sink
}

type runtimeState struct {
	aw    *asyncWriter
	sinks []*httpSink
}

func (s *runtimeState) close() {
	s.aw.Close()
	for _, k := range s.sinks {
		k.Close()
	}
}

var (
	slogger   atomic.Pointer[slog.Logger]
	state     atomic.Pointer[runtimeState]
	logLevel  = &slog.LevelVar{}
	osExit    = os.Exit
	stdoutDst io.Writer = os.Stdout
)

func init() {
	slogger.Store(slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func Init(cfg Config) error {
	if cfg.Format == FormatText && len(cfg.Sinks) > 0 {
		return errors.New("refusing to ship text-formatted logs to remote sinks; use FormatJSON")
	}

	logLevel.Set(parseLevel(cfg.Level))

	aw := newAsyncWriter(stdoutDst, asyncQueueSize)
	writers := []io.Writer{aw}
	sinks := make([]*httpSink, 0, len(cfg.Sinks))
	for _, s := range cfg.Sinks {
		hs := newHTTPSink(s)
		sinks = append(sinks, hs)
		writers = append(writers, hs)
	}

	var out io.Writer = aw
	if len(writers) > 1 {
		out = &fanout{writers: writers}
	}

	var handler slog.Handler
	if cfg.Format == FormatText {
		handler = newTextHandler(out, logLevel, cfg.Redact)
	} else {
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{
			Level:       logLevel,
			ReplaceAttr: jsonReplaceAttr(cfg.Redact),
		})
	}

	lg := slog.New(handler)
	slogger.Store(lg)
	slog.SetDefault(lg)

	if old := state.Swap(&runtimeState{aw: aw, sinks: sinks}); old != nil {
		old.close()
	}
	return nil
}

func Debug(msg string, args ...any) { slogger.Load().Debug(msg, args...) }
func Info(msg string, args ...any)  { slogger.Load().Info(msg, args...) }
func Warn(msg string, args ...any)  { slogger.Load().Warn(msg, args...) }

func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, AttrError(err))
	}
	slogger.Load().Error(msg, args...)
}

func Fatal(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, AttrError(err))
	}
	slogger.Load().Log(context.Background(), LevelFatal, msg, args...)
	Sync()
	osExit(1)
}

func SetLevel(level string) { logLevel.Set(parseLevel(level)) }

func Enabled(level string) bool { return parseLevel(level) >= logLevel.Level() }

func DebugEnabled() bool { return slog.LevelDebug >= logLevel.Level() }

func Sync() {
	if s := state.Load(); s != nil {
		s.aw.Flush()
	}
}

func Close() {
	if s := state.Swap(nil); s != nil {
		s.close()
	}
}

func parseLevel(lvl string) slog.Level {
	switch strings.ToLower(lvl) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	default:
		if lvl == "" {
			return slog.LevelInfo
		}
		return slog.LevelError
	}
}
