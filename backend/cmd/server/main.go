package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
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
	defer func() { _ = db.Close() }() // No hay mucha razón para handlear el error cuando se cierra la db
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("Database unreachable: %w", err)
	}
	logger.Info("Database connection successful")

	playerDao := dao.NewPlayerDao(nil)
	playerRepo := repository.NewPlayerRepository(playerDao)
	playerService := service.NewPlayerService(playerRepo)
	playerController := controller.NewPlayerController(playerService)

	handler := server.NewServer(
		logger,
		cfg,
		controller.NewContainer(playerController),
		nil,
	)

	server := &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info(fmt.Sprintf("Starting server at %s", cfg.GetServerAddress()))

	err = server.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("Server closed")
	} else if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}

	logger.Info("Server stopped gracefully")
	return nil
}
