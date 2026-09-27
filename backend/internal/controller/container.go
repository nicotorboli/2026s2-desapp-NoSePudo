package controller

import (
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
)

type PlayerController interface {
	GetPlayers() httphandler.Endpoint
}

type Container struct {
	Player PlayerController
}

func NewContainer(p PlayerController) *Container {
	return &Container{
		Player: p,
	}
}
