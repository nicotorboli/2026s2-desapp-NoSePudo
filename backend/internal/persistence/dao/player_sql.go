package dao

import (
	"database/sql"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerSql struct {
	Db *sql.DB
}

func (dao *PlayerSql) GetPlayer() []model.Player {
	return make([]model.Player, 1)
}

func NewPlayerDao(db *sql.DB) *PlayerSql {
	return &PlayerSql{
		Db: db,
	}
}
