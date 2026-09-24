package repository

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

type PlayerSql interface {
	GetPlayer() []model.Player
}

type PlayerRepository struct {
	sql PlayerSql
}

func (repo *PlayerRepository) GetPlayer() []model.Player {
	return repo.sql.GetPlayer()
}
