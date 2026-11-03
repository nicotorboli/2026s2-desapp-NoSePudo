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
	Market        *MarketService
	MarketMaker   *MarketMakerService
	Portfolio     *PortfolioService
	P2P           *P2PService
	P2PExpiration *P2PExpirationJob
}

// NewContainer arma el contenedor con las sincronizaciones configuradas por
// defecto y el contexto de vida que recibe. Lo usan los tests; el proceso real
// usa NewContainerWithSync.
func NewContainer(ctx context.Context, runCtx context.Context, repos *repository.Container, adapterContainer *adapters.Container, logger *slog.Logger) (*Container, error) {
	return buildContainer(ctx, runCtx, repos, adapterContainer, DefaultSyncSettings(), logger)
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

	marketMaker := NewMarketMakerService(repos.Market, time.Now)

	scoreService := NewScoreService(
		registry,
		repos.PlayerScore,
		repos.ScoreHistogram,
		repos.StrategyActivation,
		repos.LeagueCoefficient,
		repos.Player,
		repos.PlayerStatistics,
		logger,
		WithScoreQuoteReader(repos.Market),
		WithScoreQuoteWriter(marketMaker),
	)

	player := NewPlayerService(
		adapterContainer.FootballData,
		repos.Player,
		logger,
		WithMarketBootstrap(marketMaker),
		WithQuoteReader(repos.Market),
	)
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
			WithMarketAccounts(repos.Registration),
		),
		SyncRuns:      syncRuns,
		SyncScheduler: NewSyncScheduler(syncRuns, repos.SyncSchedule, settings, logger),
		Market:        NewMarketService(repos.Market, time.Now),
		MarketMaker:   marketMaker,
		Portfolio:     NewPortfolioService(repos.Portfolio),
		P2P:           NewP2PService(repos.P2P, repos.Player, time.Now),
		P2PExpiration: NewP2PExpirationJob(repos.P2P, time.Now, logger),
	}, nil
}
