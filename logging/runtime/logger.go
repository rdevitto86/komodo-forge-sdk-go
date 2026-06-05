package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var (
	slogger   *slog.Logger   = slog.New(slog.NewJSONHandler(io.Discard, nil))
	logLevel  *slog.LevelVar = &slog.LevelVar{}
	osExit    func(int)      = os.Exit
	loggerEnv string
)

func Init(name string, lvl string, env string, version string) {
	ver := version
	if ver == "" {
		ver = "unknown"
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
