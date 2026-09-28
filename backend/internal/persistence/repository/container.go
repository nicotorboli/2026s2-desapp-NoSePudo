package repository

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"

type Container struct {
	Player *PlayerRepository
}

func NewContainer(daos *dao.Container) *Container {
	return &Container{
		Player: NewPlayerRepository(daos.Player),
	}
}
