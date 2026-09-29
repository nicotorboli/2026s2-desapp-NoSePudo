package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/footballdata"
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
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting up server: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := configuration.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	appLogger := logger.Init(slog.LevelInfo)
	appLogger.Info("Initializing application...")

	var db *sql.DB
	if cfg.PostgresDataSource != "" {
		appLogger.Info("Connecting to PostgreSQL database")
		var sqlErr error
		db, sqlErr = sql.Open("postgres", cfg.PostgresDataSource)
		if sqlErr != nil {
			return fmt.Errorf("open postgres database: %w", sqlErr)
		}
		defer db.Close() //nolint:errcheck
		// Normalmente no se handlea este error en particular

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(25)
		db.SetConnMaxLifetime(5 * time.Minute)

		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if pingErr := db.PingContext(pingCtx); pingErr != nil {
			appLogger.Warn("Database ping failed on startup (continuing for stub/offline environments)", "error", pingErr.Error())
		} else {
			appLogger.Info("Database connection established successfully")
		}
	} else {
		appLogger.Warn("No database DSN provided; database connection is nil")
	}

	footballDataAdapter := footballdata.NewClient(cfg.FootballDataAPIKey, 6*time.Second)

	daoContainer := dao.NewContainer(db)
	repoContainer := repository.NewContainer(db, daoContainer)
	serviceContainer := service.NewContainer(repoContainer, footballDataAdapter, appLogger)
	controllerContainer := controller.NewContainer(serviceContainer)
	middlewareContainer := middleware.NewContainer(appLogger)

	srv := server.NewServer(cfg.GetServerAddress(), appLogger, controllerContainer, middlewareContainer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx)
}
