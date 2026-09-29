package service

import (
	"log/slog"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/footballdata"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

type Container struct {
	Player PlayerService
	Sync   PlayerSyncService
}

func NewContainer(repos *repository.Container, adapter footballdata.Client, logger *slog.Logger) *Container {
	return &Container{
		Player: NewPlayerService(repos.Player),
		Sync:   NewPlayerSyncService(adapter, repos.Player, logger),
	}
}
