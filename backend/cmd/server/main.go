package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/server"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func main() {
	if err := startServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting up server %v\n", err)
		os.Exit(1)
	}
}

func startServer() error {
	cfg := configuration.LoadCfg()
	logger := logger.NewLog()

	logger.Info("Initializing DB connection")

	db, err := sql.Open("postgres", cfg.PostgresDataSource)

	if err != nil {
		return fmt.Errorf("Invalid database credentials: %w", err)
	}
	defer db.Close() //nolint:errcheck
	// Se suprime este chequeo en particular porque no es un error que se suela handlear
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("Database unreachable: %w", err)
	}
	logger.Info("Database connection successful")

	daos := dao.NewContainer(db)
	repos := repository.NewContainer(daos)
	services := service.NewContainer(repos)
	controllers := controller.NewContainer(services)
	middlewares := middleware.NewContainer()

	srv := server.NewServer(
		logger,
		controllers,
		middlewares,
	)

	server := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      srv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info(fmt.Sprintf("Starting server at %s", cfg.GetServerAddress()))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err = <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) && err != nil {
			return fmt.Errorf("error starting server: %w", err)
		}
	case <-stop:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down server: %w", err)
		}
	}

	logger.Info("Server stopped gracefully")
	return nil
}
