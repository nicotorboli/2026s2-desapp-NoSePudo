package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
)

type Server struct {
	httpServer  *http.Server
	router      *http.ServeMux
	logger      *slog.Logger
	controllers *controller.Container
	middleware  *middleware.Container
}

func NewServer(
	addr string,
	logger *slog.Logger,
	controllers *controller.Container,
	middlewareContainer *middleware.Container,
) *Server {
	router := http.NewServeMux()
	s := &Server{
		router:      router,
		logger:      logger,
		controllers: controllers,
		middleware:  middlewareContainer,
	}

	s.routes()

	handler := middleware.Chain(
		s.router,
		middleware.CorrelationID,
		middleware.CORS,
		middleware.RequestLogger(s.logger),
	)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpServer.Handler.ServeHTTP(w, r)
}

func (s *Server) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)

	go func() {
		s.logger.Info("HTTP server listening", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen and serve: %w", err)
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		s.logger.Info("Shutting down HTTP server gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
		return nil
	}
}
