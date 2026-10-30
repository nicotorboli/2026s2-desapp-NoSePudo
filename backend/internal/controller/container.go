package controller

import (
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type Container struct {
	Player *RestPlayerController
	Sync   *RestSyncController
	Auth   *RestAuthController
}

func NewContainer(services *service.Container) *Container {
	return &Container{
		Player: NewPlayerController(services.Player),
		Sync: NewSyncController(
			services.Player,
			services.PlayerStats,
		),
		Auth: NewAuthController(services.Auth),
	}
}
