package service

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type Container struct {
	Player *Player
	Auth *Auth
}

func NewContainer(repos *repository.Container, adapterContainer *adapters.Container) *Container {
	return &Container{
		Player: NewPlayerService(repos.Player),
		Auth: NewAuthService(
			repos.User,
			adapterContainer.Password,
			adapterContainer.JWT,
			repos.RefreshToken,
			time.Now,
		),
	}
}

