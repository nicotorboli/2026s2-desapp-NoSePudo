package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/rs/zerolog"
)

type correlationIDKey struct{}

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

func Init(level slog.Level) *slog.Logger {
	zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	baseHandler := zerolog.NewSlogHandler(zl)
	handler := &contextHandler{Handler: baseHandler}
	return slog.New(handler)
}

func NewLog() *slog.Logger {
	return Init(slog.LevelInfo)
}
