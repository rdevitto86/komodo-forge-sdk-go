package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	slogger   *slog.Logger   = slog.New(slog.NewJSONHandler(io.Discard, nil))
	logLevel  *slog.LevelVar = &slog.LevelVar{}
	osExit    func(int)      = os.Exit
	initOnce  sync.Once
	loggerEnv string
)

// Configures the global logger; local/dev environments use the text handler, all others use JSON for CloudWatch.
// Must be called once at service startup; version is optional and defaults to "unknown".
func Init(name string, lvl string, env string, version ...string) {
	initOnce.Do(func() {
		ver := "unknown"
		if len(version) > 0 && version[0] != "" {
			ver = version[0]
		}

		loggerEnv = env
		logLevel.Set(effectiveLevel(lvl, env))

		var handler slog.Handler
		if isLocalEnv(env) {
			handler = NewKomodoTextHandler(os.Stdout, true, logLevel)
		} else {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
		}

		slogger = slog.New(&RedactingLogger{Handler: handler}).With(
			slog.String("service", name),
			slog.String("env", env),
			slog.String("version", ver),
		)
		slog.SetDefault(slogger)
	})
}

func Debug(msg string, args ...any) { slogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { slogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { slogger.Warn(msg, args...) }

func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, AttrError(err))
	}
	slogger.Error(msg, args...)
}

// Logs at error level and then terminates the process with status 1; deferred functions do not run.
func Fatal(msg string, err error, args ...any) {
	Error(msg, err, args...)
	osExit(1)
}

func SetLevel(level string) { logLevel.Set(effectiveLevel(level, loggerEnv)) }

func Enabled(level string) bool { return parseLevel(level) >= logLevel.Level() }

func DebugEnabled() bool { return slog.LevelDebug >= logLevel.Level() }

func isLocalEnv(env string) bool {
	e := strings.ToLower(env)
	return e == "local" || e == "dev" || e == "development"
}

func effectiveLevel(lvl, env string) slog.Level {
	level := parseLevel(lvl)
	if level < slog.LevelInfo && !isLocalEnv(env) {
		return slog.LevelInfo
	}
	return level
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
