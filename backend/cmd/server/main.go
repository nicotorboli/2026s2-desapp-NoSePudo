package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
)

type Config struct {
	env []string
}

func etc() *slog.Logger {
	return logger.NewLog()
}

func NewServer(
	logger *slog.Logger,
	config *Config,
	controllers *controller.Container) http.Handler {
	mplex := http.NewServeMux()

	return mplex
}

func main() {
	fmt.Println("Hola mundo")
}
