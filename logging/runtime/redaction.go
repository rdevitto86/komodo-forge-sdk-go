package logger

import (
	"log/slog"

	redact "github.com/rdevitto86/komodo-forge-sdk-go/security/redaction"
)

func jsonReplaceAttr(mode RedactMode) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, attr slog.Attr) slog.Attr {
		if len(groups) == 0 {
			switch attr.Key {
			case slog.LevelKey:
				if lv, ok := attr.Value.Any().(slog.Level); ok && lv >= LevelFatal {
					return slog.String(slog.LevelKey, "FATAL")
				}
				return attr
			case slog.TimeKey, slog.MessageKey, slog.SourceKey:
				return attr
			}
		}
		if mode == RedactOff {
			return attr
		}
		return redactAttr(mode, attr)
	}
}

func redactAttr(mode RedactMode, attr slog.Attr) slog.Attr {
	if mode == RedactOff {
		return attr
	}
	if redact.IsSensitiveKey(attr.Key) {
		return slog.String(attr.Key, "[REDACTED]")
	}
	if mode == RedactKeysOnly {
		return attr
	}

	v := attr.Value
	switch v.Kind() {
	case slog.KindGroup:
		sub := v.Group()
		out := make([]slog.Attr, len(sub))
		for i := range sub {
			out[i] = redactAttr(mode, sub[i])
		}
		return slog.Attr{Key: attr.Key, Value: slog.GroupValue(out...)}
	case slog.KindString:
		return slog.String(attr.Key, redact.RedactString(v.String()))
	case slog.KindAny:
		return slog.Any(attr.Key, redact.RedactValue(v.Any()))
	default:
		return attr
	}
}
