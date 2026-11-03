package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type Container struct {
	Player        *PlayerServiceImpl
	PlayerStats   *PlayerStatisticsServiceImpl
	Score         *ScoreService
	Auth          *Auth
	SyncRuns      *SyncRunService
	SyncScheduler *SyncScheduler
}

// NewContainer arma el contenedor con las sincronizaciones deshabilitadas por
// defecto. Lo usan los tests; el proceso real usa NewContainerWithSync.
func NewContainer(ctx context.Context, repos *repository.Container, adapterContainer *adapters.Container, logger *slog.Logger) (*Container, error) {
	return buildContainer(ctx, context.Background(), repos, adapterContainer, DefaultSyncSettings(), logger)
}

// NewContainerWithSync es la construcción que usa cmd/server: recibe el
// contexto de vida del proceso y la configuración de sincronizaciones.
func NewContainerWithSync(ctx context.Context, runCtx context.Context, repos *repository.Container, adapterContainer *adapters.Container, settings SyncSettings, logger *slog.Logger) (*Container, error) {
	return buildContainer(ctx, runCtx, repos, adapterContainer, settings, logger)
}

func buildContainer(ctx context.Context, runCtx context.Context, repos *repository.Container, adapterContainer *adapters.Container, settings SyncSettings, logger *slog.Logger) (*Container, error) {
	strategies, err := repos.ScoringStrategy.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load scoring strategies: %w", err)
	}

	registry := make(map[int]model.ScoringStrategy, len(strategies))
	for _, strategy := range strategies {
		registry[strategy.ID] = strategy
	}

	scoreService := NewScoreService(
		registry,
		repos.PlayerScore,
		repos.ScoreHistogram,
		repos.StrategyActivation,
		repos.LeagueCoefficient,
		repos.Player,
		repos.PlayerStatistics,
		logger,
	)

	player := NewPlayerService(adapterContainer.FootballData, repos.Player, logger)
	playerStats := NewPlayerStatisticsService(
		adapterContainer.Whoscored,
		repos.Player,
		repos.PlayerStatistics,
		scoreService,
		logger,
	)

	syncRuns := NewSyncRunService(
		runCtx,
		settings.RunTimeout,
		player,
		playerStats,
		repos.SyncRun,
		repos.SyncSchedule,
		logger,
	)

	return &Container{
		Player:      player,
		PlayerStats: playerStats,
		Score:       scoreService,
		Auth: NewAuthService(
			repos.User,
			adapterContainer.Password,
			adapterContainer.JWT,
			repos.RefreshToken,
			time.Now,
		),
		SyncRuns:      syncRuns,
		SyncScheduler: NewSyncScheduler(syncRuns, repos.SyncSchedule, settings, logger),
	}, nil
}
