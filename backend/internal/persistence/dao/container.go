package dao

import "database/sql"

type Container struct {
	Player PlayerDAO
	Audit  AuditDAO
}

func NewContainer(db *sql.DB) *Container {
	return &Container{
		Player: NewPlayerDao(db),
		Audit:  NewAuditDao(db),
	}
}
