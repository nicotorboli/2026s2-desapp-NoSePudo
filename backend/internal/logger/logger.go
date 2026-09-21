package logger

import (
	"log/slog"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewLog() *slog.Logger {
	zl := log.Logger
	handler := zerolog.NewSlogHandler(zl)
	return slog.New(handler)
}
