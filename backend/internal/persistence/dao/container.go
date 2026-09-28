package dao

import "database/sql"

type Container struct {
	Player *PlayerSql
}

func NewContainer(db *sql.DB) *Container {
	return &Container{
		Player: NewPlayerDao(db),
	}
}
