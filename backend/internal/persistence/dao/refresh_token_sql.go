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

// Las columnas del SELECT. gosec ve un nombre con "token" al lado de una
// cadena y sospecha una credencial embebida; es una lista de columnas.
const refreshTokenColumns = "id, family_id, user_id, issued_at, expires_at, used_at, revoked_at" //nolint:gosec // lista de columnas SQL, no una credencial

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

	query := "SELECT " + refreshTokenColumns + " FROM refresh_tokens WHERE id = $1"

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

// Rotate marca usada la credencial presentada e inserta su reemplazo, las dos
// cosas en una transacción (Principio XI): o queda la vieja consumida y la
// nueva viva, o no cambia nada.
//
// El UPDATE lleva las condiciones de "está viva" en su WHERE, y eso es lo que
// hace de esta operación la que decide. Dos peticiones simultáneas con la misma
// credencial compiten por la misma fila: una afecta una fila y la otra cero, y
// la que afectó cero se entera de que llegó tarde. Chequear antes con un SELECT
// no alcanzaría, porque entre el SELECT y el UPDATE las dos verían la
// credencial viva.
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
		// La fila no estaba viva: ya se usó, la revocaron, o expiró. Quien
		// llama decide qué significa mirando su estado.
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

// RevokeFamily corta una sesión y nada más. El predicado sobre user_id es
// redundante —el identificador de familia viene de una credencial ya
// verificada— y se queda como guarda barata por si un error con la clave de
// firma alguna vez lo volviera load-bearing.
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

// RevokeAllLiveForUser es la respuesta al robo: corta todas las credenciales de
// la cuenta, no sólo las de la familia afectada. Quien tiene una credencial
// robada de una familia puede tener otra, y el costo de equivocarse es un
// inicio de sesión extra.
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
