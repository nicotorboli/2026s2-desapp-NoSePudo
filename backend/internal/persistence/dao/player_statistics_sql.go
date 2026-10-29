package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerStatisticsSql struct {
	db *sql.DB
}

func NewPlayerStatisticsDao(db *sql.DB) *PlayerStatisticsSql {
	return &PlayerStatisticsSql{db: db}
}

func (d *PlayerStatisticsSql) ListByPlayerID(ctx context.Context, playerID int64) ([]model.PlayerStatistics, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `SELECT goals, assists, shots, key_passes, dribbles, tackles, rating, created_at
              FROM player_statistics WHERE player_id = $1 ORDER BY created_at DESC`

	rows, err := d.db.QueryContext(ctx, query, playerID)
	if err != nil {
		return nil, fmt.Errorf("query player statistics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	stats := make([]model.PlayerStatistics, 0)
	for rows.Next() {
		var s model.PlayerStatistics
		if err := rows.Scan(
			&s.Goals,
			&s.Assists,
			&s.Shots,
			&s.KeyPasses,
			&s.Dribbles,
			&s.Tackles,
			&s.Rating,
			&s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan player statistics: %w", err)
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("player statistics rows error: %w", err)
	}

	return stats, nil
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

	stmt, err := tx.PrepareContext(ctx, "COPY player_statistics (player_id, goals, assists, shots, key_passes, dribbles, tackles, interceptions, fouls, clears, blocks, was_dribbled, rating) FROM STDIN")
	if err != nil {
		return 0, fmt.Errorf("prepare bulk insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, row := range rows {
		if _, err := stmt.ExecContext(ctx, row.PlayerID, row.Goals, row.Assists, row.Shots, row.KeyPasses, row.Dribbles, row.Tackles, row.Interceptions, row.Fouls, row.Clears, row.Blocks, row.WasDribbled, row.Rating); err != nil {
			return 0, fmt.Errorf("stage bulk insert row for player %d: %w", row.PlayerID, err)
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		return 0, fmt.Errorf("flush bulk insert: %w", err)
	}

	return int64(len(rows)), nil
}
