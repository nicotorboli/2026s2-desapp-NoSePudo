package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// uniqueViolation es el SQLSTATE que Postgres devuelve cuando una escritura
// choca contra un índice único. Es el único lugar del código que lo conoce,
// porque es el único que habla con el driver.
const uniqueViolation = "23505"

type UserSql struct {
	Db *sql.DB
}

func NewUserDao(db *sql.DB) *UserSql {
	return &UserSql{Db: db}
}

// Insert guarda la cuenta y devuelve la copia con el id que asignó la base.
//
// El índice único sobre email es lo que realmente decide la unicidad: el
// service chequea antes para poder dar un mensaje útil, pero entre ese chequeo
// y esta escritura hay una carrera, y esto es lo que la cierra.
func (dao *UserSql) Insert(ctx context.Context, user model.User) (model.User, error) {
	if dao.Db == nil {
		return model.User{}, errNilDatabase
	}

	const query = `
		INSERT INTO users (email, password_hash, privilege, active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	// database/sql no convierte un tipo propio de una palabra por su cuenta, y
	// darle a PrivilegeLevel un driver.Valuer metería la base adentro del
	// dominio. La conversión vive acá, que es la capa que habla SQL.
	err := dao.Db.QueryRowContext(ctx, query, user.Email, user.PasswordHash, int16(user.Privilege), user.Active).
		Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		if pqErr, ok := errors.AsType[*pq.Error](err); ok && pqErr.Code == uniqueViolation {
			return model.User{}, model.ErrEmailTaken
		}
		return model.User{}, fmt.Errorf("insertar cuenta: %w", err)
	}

	return user, nil
}

// GetByEmail busca por el identificador ya normalizado. Quien llama es
// responsable de haberlo normalizado: la columna sólo contiene valores
// normalizados, así que buscar sin normalizar no encuentra nada.
func (dao *UserSql) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, privilege, active, created_at
		FROM users
		WHERE email = $1`

	return dao.queryOne(ctx, query, email)
}

func (dao *UserSql) GetByID(ctx context.Context, id int64) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, privilege, active, created_at
		FROM users
		WHERE id = $1`

	return dao.queryOne(ctx, query, id)
}

func (dao *UserSql) queryOne(ctx context.Context, query string, arg any) (model.User, error) {
	if dao.Db == nil {
		return model.User{}, errNilDatabase
	}

	var (
		user      model.User
		privilege int16
	)

	err := dao.Db.QueryRowContext(ctx, query, arg).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&privilege,
		&user.Active,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, model.ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("buscar cuenta: %w", err)
	}

	user.Privilege = privilegeFromColumn(privilege)

	return user, nil
}

// privilegeFromColumn traduce el SMALLINT al nivel de dominio enumerando los
// valores válidos en vez de convertir el entero.
//
// La diferencia no es de estilo: una conversión int16 -> uint8 trunca, y un
// valor corrupto de 258 en la columna se volvería 2, que es superusuario. Acá
// cualquier cosa que no sea exactamente uno de los dos niveles queda en
// PrivilegeUnknown, que no alcanza para nada (FR-018).
func privilegeFromColumn(value int16) model.PrivilegeLevel {
	switch value {
	case int16(model.PrivilegeUser):
		return model.PrivilegeUser
	case int16(model.PrivilegeSuperuser):
		return model.PrivilegeSuperuser
	default:
		return model.PrivilegeUnknown
	}
}
