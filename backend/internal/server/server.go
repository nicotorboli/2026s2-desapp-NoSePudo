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
		router:      http.NewServeMux(),
		logger:      logger,
		controllers: controllers,
		middleware:  middleware,
	}

	s.routes()

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
