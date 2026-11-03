package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

const uniqueViolation = "23505"

type UserSql struct {
	db *sql.DB
}

func NewUserDao(db *sql.DB) *UserSql {
	return &UserSql{db: db}
}

func (d *UserSql) Insert(ctx context.Context, user model.User) (model.User, error) {
	if d.db == nil {
		return model.User{}, errNilDatabase
	}

	const query = `
		INSERT INTO users (email, password_hash, privilege, active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := d.db.QueryRowContext(ctx, query, user.Email, user.PasswordHash, int16(user.Privilege), user.Active).
		Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		if pqErr, ok := errors.AsType[*pq.Error](err); ok && pqErr.Code == uniqueViolation {
			return model.User{}, model.ErrEmailTaken
		}
		return model.User{}, fmt.Errorf("insertar cuenta: %w", err)
	}

	return user, nil
}

func (d *UserSql) InsertTx(ctx context.Context, tx *sql.Tx, user model.User) (model.User, error) {
	const query = `
		INSERT INTO users (email, password_hash, privilege, active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	querier, err := querierFor(tx, d.db)
	if err != nil {
		return model.User{}, err
	}

	err = querier.QueryRowContext(ctx, query, user.Email, user.PasswordHash, int16(user.Privilege), user.Active).
		Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		if pqErr, ok := errors.AsType[*pq.Error](err); ok && pqErr.Code == uniqueViolation {
			return model.User{}, model.ErrEmailTaken
		}
		return model.User{}, fmt.Errorf("insertar cuenta: %w", err)
	}

	return user, nil
}

func (d *UserSql) GetSuperuser(ctx context.Context) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, privilege, active, created_at
		FROM users
		WHERE privilege = $1
		ORDER BY id
		LIMIT 1`

	return d.queryOne(ctx, query, int16(model.PrivilegeSuperuser))
}

func (d *UserSql) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, privilege, active, created_at
		FROM users
		WHERE email = $1`

	return d.queryOne(ctx, query, email)
}

func (d *UserSql) GetByID(ctx context.Context, id int64) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, privilege, active, created_at
		FROM users
		WHERE id = $1`

	return d.queryOne(ctx, query, id)
}

func (d *UserSql) queryOne(ctx context.Context, query string, arg any) (model.User, error) {
	if d.db == nil {
		return model.User{}, errNilDatabase
	}

	var (
		user      model.User
		privilege int16
	)

	err := d.db.QueryRowContext(ctx, query, arg).Scan(
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
