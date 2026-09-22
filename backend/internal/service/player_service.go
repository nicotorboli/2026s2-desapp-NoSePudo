package service

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

type PlayerRepository interface {
	GetPlayers() []model.Player
}

type PlayerService struct {
	playerRepo PlayerRepository
}

func (p *PlayerService) ListPlayers() []model.Player {
	return p.playerRepo.GetPlayers()
}
