package controller

import (
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type Container struct {
	Player  *RestPlayerController
	Sync    *RestSyncController
	Scoring *RestScoringController
	Auth    *RestAuthController
	Market  *RestMarketController
	P2P     *RestP2PController
}

func NewContainer(services *service.Container) *Container {
	return &Container{
		Player: NewPlayerController(services.Player),
		Sync: NewSyncController(
			services.PlayerStats,
			services.SyncRuns,
		),
		Scoring: NewScoringController(services.Score, WithQuoteService(services.Score)),
		Auth:    NewAuthController(services.Auth),
		Market:  NewMarketController(services.Market, services.Portfolio),
		P2P:     NewP2PController(services.P2P),
	}
}
