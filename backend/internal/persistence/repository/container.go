package repository

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"

type Container struct {
	Player *PlayerRepository
	User *UserRepository
	RefreshToken *RefreshTokenRepository
}

func NewContainer(daos *dao.Container) *Container {
	return &Container{
		Player: NewPlayerRepository(daos.Player),
		User: NewUserRepository(daos.User),
		RefreshToken: NewRefreshTokenRepository(daos.RefreshToken),
	}
}

