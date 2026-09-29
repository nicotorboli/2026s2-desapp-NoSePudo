package repository

import (
	"context"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// UserSql es lo que el repositorio necesita de la persistencia. La interfaz se
// declara acá, del lado de quien la consume, así que el repositorio se puede
// testear sin base y la implementación concreta queda abajo.
type UserSql interface {
	Insert(ctx context.Context, user model.User) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
	GetByID(ctx context.Context, id int64) (model.User, error)
}

// UserRepository es el repositorio de la cuenta: un concepto de dominio, no
// una tecnología. Quien lo usa piensa en cuentas y no en tablas.
type UserRepository struct {
	sql UserSql
}

func NewUserRepository(sql UserSql) *UserRepository {
	return &UserRepository{sql: sql}
}

func (repo *UserRepository) Insert(ctx context.Context, user model.User) (model.User, error) {
	return repo.sql.Insert(ctx, user)
}

func (repo *UserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	return repo.sql.GetByEmail(ctx, email)
}

func (repo *UserRepository) GetByID(ctx context.Context, id int64) (model.User, error) {
	return repo.sql.GetByID(ctx, id)
}
