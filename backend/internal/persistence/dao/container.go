package dao

import (
	"database/sql"
	"errors"
)

// errNilDatabase es la guarda compartida por los DAOs: sin conexión no hay
// consulta que hacer, y conviene decirlo acá y no dejar que explote adentro
// del driver.
var errNilDatabase = errors.New("la conexión a la base de datos es nil")

type Container struct {
	Player *PlayerSql
	User *UserSql
	RefreshToken *RefreshTokenSql
	Schema *SchemaSql
}

func NewContainer(db *sql.DB) *Container {
	return &Container{
		Player: NewPlayerDao(db),
		User: NewUserDao(db),
		RefreshToken: NewRefreshTokenDao(db),
		Schema: NewSchemaDao(db),
	}
}

