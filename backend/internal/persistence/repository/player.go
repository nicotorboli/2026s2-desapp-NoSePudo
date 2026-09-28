package repository

import (
	"context"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerSql interface {
	GetPlayer(ctx context.Context) ([]model.Player, error)
}

type PlayerRepository struct {
	sql PlayerSql
}

func (repo *PlayerRepository) GetPlayer(ctx context.Context) ([]model.Player, error) {
	return repo.sql.GetPlayer(ctx)
}

func NewPlayerRepository(sql PlayerSql) *PlayerRepository {
	return &PlayerRepository{
		sql: sql,
	}
}
