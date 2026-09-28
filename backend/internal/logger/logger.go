package logger

import (
	"log/slog"
	"os"

	"github.com/rs/zerolog"
)

func NewLog() *slog.Logger {
	zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	handler := zerolog.NewSlogHandler(zl)
	return slog.New(handler)
}
