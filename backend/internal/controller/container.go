package controller

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"

type Container struct {
	Player *PlayerController
	Sync   *SyncController
}

func NewContainer(services *service.Container) *Container {
	return &Container{
		Player: NewPlayerController(services.Player),
		Sync:   NewSyncController(services.Sync),
	}
}
