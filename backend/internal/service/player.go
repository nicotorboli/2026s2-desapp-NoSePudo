package service

import (
	"context"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerRepository interface {
	GetPlayer(ctx context.Context) ([]model.Player, error)
}

type Player struct {
	repo PlayerRepository
}

func (p *Player) ListPlayers(ctx context.Context) ([]model.Player, error) {
	return p.repo.GetPlayer(ctx)
}

func NewPlayerService(r PlayerRepository) *Player {
	return &Player{
		repo: r,
	}
}
