package controller

import (
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type PlayerController interface {
	GetPlayers() httphandler.Endpoint
}

type Container struct {
	Player PlayerController
}

func NewContainer(services *service.Container) *Container {
	return &Container{
		Player: NewPlayerController(services.Player),
	}
}
