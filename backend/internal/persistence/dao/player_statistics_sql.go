package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerStatisticsDAO interface {
	InsertStatistics(ctx context.Context, tx *sql.Tx, rows []model.PlayerStatistics) (int64, error)
}

type PlayerStatisticsSql struct {
	db *sql.DB
}

func NewPlayerStatisticsDao(db *sql.DB) *PlayerStatisticsSql {
	return &PlayerStatisticsSql{db: db}
}

func (d *PlayerStatisticsSql) InsertStatistics(ctx context.Context, tx *sql.Tx, rows []model.PlayerStatistics) (int64, error) {
	if d.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}
	if tx == nil {
		return 0, fmt.Errorf("transaction is required")
	}
	if len(rows) == 0 {
		return 0, nil
	}

	stmt, err := tx.PrepareContext(ctx, "COPY player_statistics (player_id, goals, assists, shots, key_passes, dribbles, tackles, rating) FROM STDIN")
	if err != nil {
		return 0, fmt.Errorf("prepare bulk insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, row := range rows {
		if _, err := stmt.ExecContext(ctx, row.PlayerID, row.Goals, row.Assists, row.Shots, row.KeyPasses, row.Dribbles, row.Tackles, row.Rating); err != nil {
			return 0, fmt.Errorf("stage bulk insert row for player %d: %w", row.PlayerID, err)
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		return 0, fmt.Errorf("flush bulk insert: %w", err)
	}

	return int64(len(rows)), nil
}
