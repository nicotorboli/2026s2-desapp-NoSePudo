package server

import (
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
)

func addRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	controllers *controller.Container,
	middleware *middleware.Container,
) {

	mux.Handle("GET /player/", httphandler.Wrap(controllers.Player.GetPlayers(), logger))

}
