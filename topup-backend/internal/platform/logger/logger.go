// Package logger menyiapkan slog JSON dengan masking data sensitif (RULES §3.6).
package logger

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// sensitiveKeys: atribut dengan nama ini (atau mengandungnya) selalu dimasking.
var sensitiveKeys = []string{
	"password", "token", "secret", "authorization", "credential",
	"serial_number", "api_key", "otp", "totp", "recovery_code",
}

const masked = "***"

// New membuat logger JSON. level: debug|info|warn|error.
func New(w io.Writer, level, appEnv, service string) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: maskAttr,
	})
	return slog.New(h).With("service", service, "env", appEnv)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
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

func maskAttr(_ []string, a slog.Attr) slog.Attr {
	if IsSensitive(a.Key) {
		return slog.String(a.Key, masked)
	}
	return a
}

// IsSensitive melaporkan apakah nama field harus dimasking.
func IsSensitive(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}

type ctxKey struct{}

// WithContext menyisipkan logger (biasanya sudah membawa request_id) ke context.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext mengambil logger dari context, atau slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
