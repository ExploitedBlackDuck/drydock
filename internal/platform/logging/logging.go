// Package logging configures Drydock's operational logger (log/slog). This is
// distinct from the durable, hash-chained audit log (PROJECT-BOOK §2.4): slog
// output is for the operator and developer and is rotated and disposable.
//
// Secrets must never reach the log. Use the Sensitive* attribute constructors,
// or name attributes with a sensitive key, and their values are redacted at the
// logging boundary before they are written.
package logging

import (
	"io"
	"log/slog"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Redacted is the placeholder substituted for any value carrying a secret.
const Redacted = "[REDACTED]"

// sensitiveKeys are attribute keys whose values are always redacted, regardless
// of how they are logged. Matching is case-insensitive.
var sensitiveKeys = map[string]struct{}{
	"passphrase":  {},
	"password":    {},
	"secret":      {},
	"token":       {},
	"private_key": {},
	"data_key":    {},
	"tls_key":     {},
	"key_ref":     {},
}

// Options configures the operational logger.
type Options struct {
	// Level is the minimum level emitted ("debug", "info", "warn", "error").
	Level string
	// Format is "json" for structured file output or "text" for pretty
	// development output.
	Format string
}

// New builds an *slog.Logger writing to w, redacting sensitive attributes. It
// never returns nil; an unrecognized level falls back to info and an
// unrecognized format falls back to text.
func New(w io.Writer, opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{
		Level:       parseLevel(opts.Level),
		ReplaceAttr: redactAttr,
	}

	var handler slog.Handler
	if strings.EqualFold(opts.Format, "json") {
		handler = slog.NewJSONHandler(w, handlerOpts)
	} else {
		handler = slog.NewTextHandler(w, handlerOpts)
	}
	return slog.New(handler)
}

// RotatingFile returns an io.WriteCloser that writes to path and rotates it,
// keeping a bounded amount of history on disk.
func RotatingFile(path string) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// redactAttr is the slog ReplaceAttr hook. It masks the value of any attribute
// whose key is sensitive, recursing into groups.
func redactAttr(_ []string, a slog.Attr) slog.Attr {
	if _, ok := sensitiveKeys[strings.ToLower(a.Key)]; ok {
		a.Value = slog.StringValue(Redacted)
	}
	return a
}

// Secret wraps a known-secret string as an slog attribute whose value is always
// redacted, regardless of its key. Prefer this when logging a value that is
// secret by nature rather than by name.
func Secret(key, _ string) slog.Attr {
	return slog.String(key, Redacted)
}

// Redact returns s with its value masked. Use it when interpolating a secret is
// unavoidable; prefer structured attributes.
func Redact(_ string) string {
	return Redacted
}
