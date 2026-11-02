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
	Player      *PlayerServiceImpl
	PlayerStats *PlayerStatisticsServiceImpl
	Score       *ScoreService
	Auth        *Auth
}

func NewContainer(ctx context.Context, repos *repository.Container, adapterContainer *adapters.Container, logger *slog.Logger) (*Container, error) {
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

	return &Container{
		Player: NewPlayerService(adapterContainer.FootballData, repos.Player, logger),
		PlayerStats: NewPlayerStatisticsService(
			adapterContainer.Whoscored,
			repos.Player,
			repos.PlayerStatistics,
			scoreService,
			logger,
		),
		Score: scoreService,
		Auth: NewAuthService(
			repos.User,
			adapterContainer.Password,
			adapterContainer.JWT,
			repos.RefreshToken,
			time.Now,
		),
	}, nil
}
