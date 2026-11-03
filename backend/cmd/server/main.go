package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
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

// expectedTables son las tablas sin las cuales el servicio no puede trabajar.
// La lista y la decisión de abortar viven acá y no en el DAO: el DAO sólo
// corre la consulta.
var expectedTables = []string{"players", "player_statistics", "player_audit_logs", "users", "refresh_tokens", "scoring_strategies", "scoring_strategy_activations", "scoring_histograms", "player_scores", "league_coefficients", "sync_schedules", "sync_runs", "user_accounts", "user_positions", "market_orders", "market_trades", "ledger_entries", "player_quotes"}

// ensureSchema se niega a arrancar cuando falta alguna tabla, por la misma
// razón por la que el servicio se niega a arrancar sin secreto de firma: es
// una precondición de infraestructura, y conviene fallar temprano y claro en
// vez de tarde y confuso.
//
// El síntoma que evita es concreto: init.sql sólo corre cuando el volumen de
// Postgres está vacío, así que editarlo no cambia nada hasta recrear el
// volumen. Sin esto, quien hace git pull y arranca se come un 500 con
// 'relation "users" does not exist' desde tres capas abajo, sin ninguna pista
// de que lo que cambió fue un .sql que su base nunca leyó.
func ensureSchema(ctx context.Context, schema *dao.SchemaSql) error {
	missing, err := schema.MissingTables(ctx, expectedTables)
	if err != nil {
		return fmt.Errorf("verificar el esquema de la base: %w", err)
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"faltan tablas en la base: %s\n"+
				"db/init.sql sólo se ejecuta cuando el volumen de Postgres está vacío, "+
				"así que editarlo no cambia nada hasta recrear el volumen.\n"+
				"Desde backend/: docker compose down -v && docker compose up -d",
			strings.Join(missing, ", "),
		)
	}

	return nil
}

func startServer() error {
	cfg, err := configuration.LoadCfg()
	if err != nil {
		return fmt.Errorf("configuración inválida: %w", err)
	}

	logger := logger.NewLog()

	logger.Info("Initializing DB connection")

	db, err := sql.Open("postgres", cfg.PostgresDataSource)

	if err != nil {
		return fmt.Errorf("Invalid database credentials: %w", err)
	}
	defer func() { _ = db.Close() }()
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

	if err = ensureSchema(ctx, daos.Schema); err != nil {
		return err
	}
	logger.Info("Database schema verified")

	repos := repository.NewContainer(db, daos,
		repository.WithInitialUserBalance(model.CreditsFromUnits(cfg.InitialUserBalance)),
	)
	adapterContainer := adapters.NewContainer(cfg)

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	services, err := service.NewContainerWithSync(
		ctx,
		runCtx,
		repos,
		adapterContainer,
		service.SyncSettings{
			RunTimeout:         cfg.SyncRunTimeout,
			PlayersInterval:    cfg.SyncPlayersInterval,
			StatisticsInterval: cfg.SyncStatisticsInterval,
			PlayersEnabled:     cfg.SyncPlayersEnabled,
			StatisticsEnabled:  cfg.SyncStatisticsEnabled,
		},
		logger,
	)

	if err != nil {
		return err
	}
	if cfg.HasSuperuserCredentials() {
		if err = services.Auth.EnsureSuperuser(ctx, cfg.SuperuserEmail, cfg.SuperuserPassword); err != nil {
			return fmt.Errorf("aprovisionar el superusuario: %w", err)
		}
		logger.Info("Superuser account ensured")

		superuser, superuserErr := services.Auth.Superuser(ctx, cfg.SuperuserEmail)
		if superuserErr != nil {
			return fmt.Errorf("obtener el superusuario: %w", superuserErr)
		}
		if activationErr := services.Score.EnsureDefaultActivation(ctx, superuser.ID); activationErr != nil {
			return fmt.Errorf("activar la estrategia por defecto: %w", activationErr)
		}
		logger.Info("Default scoring strategy ensured")

		if err = services.MarketMaker.Bootstrap(ctx); err != nil {
			return fmt.Errorf("preparar el mercado: %w", err)
		}
		logger.Info("Market bootstrapped")
	}

	controllers := controller.NewContainer(services)
	middlewares := middleware.NewContainer(adapterContainer.JWT)

	contractPath := filepath.Join("api", "openapi.yaml")

	srv := server.NewServer(cfg.GetServerAddress(), logger, controllers, middlewares, contractPath)

	services.SyncScheduler.Start(runCtx)

	err = srv.Run(runCtx)

	stop()
	services.SyncScheduler.Stop()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if shutdownErr := services.SyncRuns.Shutdown(shutdownCtx); shutdownErr != nil {
		logger.Warn("sync runs did not finish before shutdown", "error", shutdownErr.Error())
	}

	return err
}
