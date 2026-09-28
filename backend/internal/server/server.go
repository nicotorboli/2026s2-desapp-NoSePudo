package server

import (
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
)

type Server struct {
	router      *http.ServeMux
	logger      *slog.Logger
	controllers *controller.Container
	middleware  *middleware.Container
}

func NewServer(
	logger *slog.Logger,
	controllers *controller.Container,
	middleware *middleware.Container,
) *Server {
	s := &Server{
		logger:      logger,
		controllers: controllers,
		middleware:  middleware,
	}

	// El router no existe hasta que buildMux corre sobre la descripción que
	// devuelve routes(). Mientras tanto no hay ningún mux en alcance al que
	// registrarle una ruta por fuera de la tabla.
	s.router = buildMux(s.routes(), s.logger, s.middleware)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
