package repository

import (
	"database/sql"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type Container struct {
	Player PlayerRepository
}

func NewContainer(db *sql.DB, daos *dao.Container) *Container {
	return &Container{
		Player: NewPlayerRepository(db, daos.Player, daos.Audit),
	}
}
