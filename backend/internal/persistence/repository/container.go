package repository

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"

type Container struct {
	Player *PlayerRepository
	User   *UserRepository
}

func NewContainer(daos *dao.Container) *Container {
	return &Container{
		Player: NewPlayerRepository(daos.Player),
		User:   NewUserRepository(daos.User),
	}
}
