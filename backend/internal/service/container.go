package service

import (
	"log/slog"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type Container struct {
	Player      *PlayerServiceImpl
	PlayerStats *PlayerStatisticsServiceImpl
	Auth        *Auth
}

func NewContainer(repos *repository.Container, adapterContainer *adapters.Container, logger *slog.Logger) *Container {
	return &Container{
		Player: NewPlayerService(adapterContainer.FootballData, repos.Player, logger),
		PlayerStats: NewPlayerStatisticsService(
			adapterContainer.Whoscored,
			repos.Player,
			repos.PlayerStatistics,
			logger,
		),
		Auth: NewAuthService(
			repos.User,
			adapterContainer.Password,
			adapterContainer.JWT,
			repos.RefreshToken,
			time.Now,
		),
	}
}
