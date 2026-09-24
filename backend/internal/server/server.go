package server

import (
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/cfg"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
)

func NewServer(
	logger *slog.Logger,
	config *cfg.Config,
	controllers *controller.Container) http.Handler {
	mplex := http.NewServeMux()

	return mplex
}
