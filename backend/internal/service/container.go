package service

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"

type Container struct {
	Player *Player
}

func NewContainer(repos *repository.Container) *Container {
	return &Container{
		Player: NewPlayerService(repos.Player),
	}
}
