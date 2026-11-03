package controller

import (
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type Container struct {
	Player  *RestPlayerController
	Sync    *RestSyncController
	Scoring *RestScoringController
	Auth    *RestAuthController
}

func NewContainer(services *service.Container) *Container {
	return &Container{
		Player: NewPlayerController(services.Player),
		Sync: NewSyncController(
			services.PlayerStats,
			services.SyncRuns,
		),
		Scoring: NewScoringController(services.Score),
		Auth:    NewAuthController(services.Auth),
	}
}
