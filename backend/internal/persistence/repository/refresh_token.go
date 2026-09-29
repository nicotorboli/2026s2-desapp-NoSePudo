package repository

import (
	"context"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// RefreshTokenSql es lo que el repositorio necesita de la persistencia.
type RefreshTokenSql interface {
	Insert(ctx context.Context, token model.RefreshToken) error
	GetByID(ctx context.Context, id string) (model.RefreshToken, error)
	Rotate(ctx context.Context, presentedID string, replacement model.RefreshToken) error
	RevokeFamily(ctx context.Context, familyID string, userID int64) error
	RevokeAllLiveForUser(ctx context.Context, userID int64) error
}

// RefreshTokenRepository es el repositorio de la credencial de renovación: un
// concepto de dominio, no una tecnología.
type RefreshTokenRepository struct {
	sql RefreshTokenSql
}

func NewRefreshTokenRepository(sql RefreshTokenSql) *RefreshTokenRepository {
	return &RefreshTokenRepository{sql: sql}
}

func (repo *RefreshTokenRepository) Insert(ctx context.Context, token model.RefreshToken) error {
	return repo.sql.Insert(ctx, token)
}

func (repo *RefreshTokenRepository) GetByID(ctx context.Context, id string) (model.RefreshToken, error) {
	return repo.sql.GetByID(ctx, id)
}

func (repo *RefreshTokenRepository) Rotate(
	ctx context.Context,
	presentedID string,
	replacement model.RefreshToken,
) error {
	return repo.sql.Rotate(ctx, presentedID, replacement)
}

func (repo *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID string, userID int64) error {
	return repo.sql.RevokeFamily(ctx, familyID, userID)
}

func (repo *RefreshTokenRepository) RevokeAllLiveForUser(ctx context.Context, userID int64) error {
	return repo.sql.RevokeAllLiveForUser(ctx, userID)
}
