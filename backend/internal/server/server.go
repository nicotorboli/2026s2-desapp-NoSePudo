package server

import (
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
)

func NewServer(
	logger *slog.Logger,
	cfg *configuration.Cfg,
	controllers *controller.Container,
	middleware *middleware.Container,
) http.Handler {
	mux := http.NewServeMux()

	addRoutes(mux, logger, controllers, middleware)

	return mux
}
