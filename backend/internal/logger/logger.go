package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/rs/zerolog"
)

type correlationIDKey struct{}

// CorrelationIDContextKey is the context key for correlation ID.
var CorrelationIDContextKey = correlationIDKey{}

type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if correlationID, ok := ctx.Value(CorrelationIDContextKey).(string); ok && correlationID != "" {
			r.AddAttrs(slog.String("correlation_id", correlationID))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}

// Init creates a new structured JSON logger using zerolog as the underlying slog handler.
func Init(level slog.Level) *slog.Logger {
	zl := zerolog.New(os.Stdout).With().Timestamp().Logger()
	baseHandler := zerolog.NewSlogHandler(zl)
	handler := &contextHandler{Handler: baseHandler}
	return slog.New(handler)
}

// NewLog returns a default initialized logger.
func NewLog() *slog.Logger {
	return Init(slog.LevelInfo)
}
