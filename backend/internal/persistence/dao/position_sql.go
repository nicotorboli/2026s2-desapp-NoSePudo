package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PositionSql struct {
	Db *sql.DB
}

func NewPositionDao(db *sql.DB) *PositionSql {
	return &PositionSql{Db: db}
}

func (dao *PositionSql) InsertIfAbsent(ctx context.Context, tx *sql.Tx, position model.Position) (bool, error) {
	const query = `
		INSERT INTO user_positions (user_id, player_id, available_quantity, reserved_quantity, cost_basis)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, player_id) DO NOTHING`

	result, err := execOn(ctx, tx, dao.Db, query,
		position.UserID, position.PlayerID,
		position.AvailableQuantity, position.ReservedQuantity, int64(position.CostBasis),
	)
	if err != nil {
		return false, fmt.Errorf("insertar la posición: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("contar las filas afectadas: %w", err)
	}

	return affected == 1, nil
}

func (dao *PositionSql) LockByUserAndPlayer(ctx context.Context, tx *sql.Tx, userID, playerID int64) (model.Position, error) {
	const query = `
		SELECT user_id, player_id, available_quantity, reserved_quantity, cost_basis
		FROM user_positions
		WHERE user_id = $1 AND player_id = $2
		FOR UPDATE`

	if tx == nil {
		return model.Position{}, errLockNeedsTransaction
	}

	return scanPosition(tx.QueryRowContext(ctx, query, userID, playerID))
}

func (dao *PositionSql) Update(ctx context.Context, tx *sql.Tx, position model.Position) error {
	const query = `
		UPDATE user_positions
		SET available_quantity = $3, reserved_quantity = $4, cost_basis = $5, updated_at = NOW()
		WHERE user_id = $1 AND player_id = $2`

	result, err := execOn(ctx, tx, dao.Db, query,
		position.UserID, position.PlayerID,
		position.AvailableQuantity, position.ReservedQuantity, int64(position.CostBasis),
	)
	if err != nil {
		return fmt.Errorf("actualizar la posición: %w", err)
	}

	return requireOneRow(result, model.ErrPositionNotFound)
}

func (dao *PositionSql) ListByUser(ctx context.Context, userID int64) ([]model.Position, error) {
	const query = `
		SELECT user_id, player_id, available_quantity, reserved_quantity, cost_basis
		FROM user_positions
		WHERE user_id = $1
		ORDER BY player_id`

	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listar las posiciones: %w", err)
	}
	defer func() { _ = rows.Close() }()

	positions := make([]model.Position, 0)
	for rows.Next() {
		position, scanErr := scanPositionFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer las posiciones: %w", err)
	}

	return positions, nil
}

func (dao *PositionSql) TotalQuantityByPlayer(ctx context.Context, playerID int64) (int64, error) {
	const query = `
		SELECT COALESCE(SUM(available_quantity + reserved_quantity), 0)
		FROM user_positions
		WHERE player_id = $1`

	if dao.Db == nil {
		return 0, errNilDatabase
	}

	var total int64
	if err := dao.Db.QueryRowContext(ctx, query, playerID).Scan(&total); err != nil {
		return 0, fmt.Errorf("sumar los tokens del jugador: %w", err)
	}

	return total, nil
}

func scanPosition(row *sql.Row) (model.Position, error) {
	var (
		position  model.Position
		costBasis int64
	)

	err := row.Scan(
		&position.UserID,
		&position.PlayerID,
		&position.AvailableQuantity,
		&position.ReservedQuantity,
		&costBasis,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Position{}, model.ErrPositionNotFound
		}
		return model.Position{}, fmt.Errorf("leer la posición: %w", err)
	}

	position.CostBasis = model.Credits(costBasis)

	return position, nil
}

func scanPositionFromRows(rows *sql.Rows) (model.Position, error) {
	var (
		position  model.Position
		costBasis int64
	)

	err := rows.Scan(
		&position.UserID,
		&position.PlayerID,
		&position.AvailableQuantity,
		&position.ReservedQuantity,
		&costBasis,
	)
	if err != nil {
		return model.Position{}, fmt.Errorf("leer la posición: %w", err)
	}

	position.CostBasis = model.Credits(costBasis)

	return position, nil
}
