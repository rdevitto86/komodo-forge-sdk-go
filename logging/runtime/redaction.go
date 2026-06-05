package logger

import (
	"context"
	"log/slog"

	"github.com/rdevitto86/komodo-forge-sdk-go/api/redaction"
)

type RedactingLogger struct{ slog.Handler }

func (rl *RedactingLogger) Handle(ctx context.Context, rec slog.Record) error {
	clean := slog.NewRecord(rec.Time, rec.Level, rec.Message, rec.PC)
	rec.Attrs(func(attr slog.Attr) bool {
		clean.AddAttrs(slog.Any(attr.Key, redaction.RedactPair(attr.Key, attr.Value.Any())))
		return true
	})
	return rl.Handler.Handle(ctx, clean)
}
