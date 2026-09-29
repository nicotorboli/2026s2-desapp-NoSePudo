package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type RefreshTokenSql struct {
	Db *sql.DB
}

func NewRefreshTokenDao(db *sql.DB) *RefreshTokenSql {
	return &RefreshTokenSql{Db: db}
}

func (dao *RefreshTokenSql) Insert(ctx context.Context, token model.RefreshToken) error {
	if dao.Db == nil {
		return errNilDatabase
	}

	const query = `
		INSERT INTO refresh_tokens (id, family_id, user_id, expires_at)
		VALUES ($1, $2, $3, $4)`

	if _, err := dao.Db.ExecContext(ctx, query, token.ID, token.FamilyID, token.UserID, token.ExpiresAt); err != nil {
		return fmt.Errorf("insertar la credencial de renovación: %w", err)
	}

	return nil
}

func (dao *RefreshTokenSql) GetByID(ctx context.Context, id string) (model.RefreshToken, error) {
	if dao.Db == nil {
		return model.RefreshToken{}, errNilDatabase
	}

	const query = `
		SELECT id, family_id, user_id, issued_at, expires_at, used_at, revoked_at
		FROM refresh_tokens
		WHERE id = $1`

	var token model.RefreshToken
	err := dao.Db.QueryRowContext(ctx, query, id).Scan(
		&token.ID,
		&token.FamilyID,
		&token.UserID,
		&token.IssuedAt,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.RefreshToken{}, model.ErrRefreshTokenNotFound
		}
		return model.RefreshToken{}, fmt.Errorf("buscar la credencial de renovación: %w", err)
	}

	return token, nil
}

func (dao *RefreshTokenSql) Rotate(ctx context.Context, presentedID string, replacement model.RefreshToken) error {
	if dao.Db == nil {
		return errNilDatabase
	}

	transaction, err := dao.Db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir la transacción de rotación: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	const markUsed = `
		UPDATE refresh_tokens
		SET used_at = NOW()
		WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL AND expires_at > NOW()`

	result, err := transaction.ExecContext(ctx, markUsed, presentedID)
	if err != nil {
		return fmt.Errorf("marcar usada la credencial de renovación: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("leer el resultado de la rotación: %w", err)
	}
	if affected == 0 {

		return model.ErrRefreshTokenReused
	}

	const insertReplacement = `
		INSERT INTO refresh_tokens (id, family_id, user_id, expires_at)
		VALUES ($1, $2, $3, $4)`

	_, err = transaction.ExecContext(ctx, insertReplacement,
		replacement.ID, replacement.FamilyID, replacement.UserID, replacement.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insertar la credencial de reemplazo: %w", err)
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("confirmar la rotación: %w", err)
	}

	return nil
}

func (dao *RefreshTokenSql) RevokeFamily(ctx context.Context, familyID string, userID int64) error {
	if dao.Db == nil {
		return errNilDatabase
	}

	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE family_id = $1 AND user_id = $2 AND revoked_at IS NULL`

	if _, err := dao.Db.ExecContext(ctx, query, familyID, userID); err != nil {
		return fmt.Errorf("revocar la familia de sesión: %w", err)
	}

	return nil
}

func (dao *RefreshTokenSql) RevokeAllLiveForUser(ctx context.Context, userID int64) error {
	if dao.Db == nil {
		return errNilDatabase
	}

	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`

	if _, err := dao.Db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("revocar las credenciales de la cuenta: %w", err)
	}

	return nil
}
