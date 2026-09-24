package service

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

type PlayerRepository interface {
	GetPlayers() []model.Player
}

type Player struct {
	repo PlayerRepository
}

func (p *Player) ListPlayers() []model.Player {
	return p.repo.GetPlayers()
}
