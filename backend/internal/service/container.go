package service

import (
	"log/slog"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type Container struct {
	Player      PlayerService
	Sync        PlayerSyncService
	Auth        *Auth
	Market      *MarketService
	MarketMaker *MarketMakerService
}

func NewContainer(repos *repository.Container, adapterContainer *adapters.Container, logger *slog.Logger) *Container {
	marketMaker := NewMarketMakerService(repos.Market, time.Now)

	return &Container{
		Player: NewPlayerService(repos.Player),
		Sync:   NewPlayerSyncService(adapterContainer.FootballData, repos.Player, logger, WithMarketBootstrap(marketMaker)),
		Auth: NewAuthService(
			repos.User,
			adapterContainer.Password,
			adapterContainer.JWT,
			repos.RefreshToken,
			time.Now,
			WithMarketAccounts(repos.Registration),
		),
		Market:      NewMarketService(repos.Market, time.Now),
		MarketMaker: marketMaker,
	}
}
