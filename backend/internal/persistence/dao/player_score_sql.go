package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerScoreSql struct {
	db *sql.DB
}

func NewPlayerScoreDao(db *sql.DB) *PlayerScoreSql {
	return &PlayerScoreSql{db: db}
}

func (d *PlayerScoreSql) Insert(ctx context.Context, tx *sql.Tx, scores []model.PlayerScore) error {
	if d.db == nil {
		return errNilDatabase
	}
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}
	if len(scores) == 0 {
		return nil
	}

	const query = `INSERT INTO player_scores (player_id, statistics_id, strategy_id, weights, raw_score, percentile, histogram_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare insert player score: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, s := range scores {
		weights, err := json.Marshal(s.Weights)
		if err != nil {
			return fmt.Errorf("marshal weights: %w", err)
		}
		if _, err := stmt.ExecContext(ctx, s.PlayerID, s.StatisticsID, s.StrategyID, weights, s.RawScore, s.Percentile, s.HistogramID); err != nil {
			return fmt.Errorf("insert player score for player %d: %w", s.PlayerID, err)
		}
	}

	return nil
}

func (d *PlayerScoreSql) ExistsFor(ctx context.Context, playerID int64, strategyID int, statisticsID int64) (bool, error) {
	if d.db == nil {
		return false, errNilDatabase
	}

	const query = `SELECT EXISTS(SELECT 1 FROM player_scores WHERE player_id = $1 AND strategy_id = $2 AND statistics_id = $3)`

	var exists bool
	if err := d.db.QueryRowContext(ctx, query, playerID, strategyID, statisticsID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check player score existence: %w", err)
	}

	return exists, nil
}

func (d *PlayerScoreSql) ListByPlayer(ctx context.Context, playerID int64) ([]model.PlayerScore, error) {
	if d.db == nil {
		return nil, errNilDatabase
	}

	const query = `SELECT id, player_id, statistics_id, strategy_id, weights, raw_score, percentile, histogram_id, computed_at
		FROM player_scores WHERE player_id = $1 ORDER BY computed_at DESC, id DESC`

	rows, err := d.db.QueryContext(ctx, query, playerID)
	if err != nil {
		return nil, fmt.Errorf("query player scores: %w", err)
	}
	defer func() { _ = rows.Close() }()

	scores := make([]model.PlayerScore, 0)
	for rows.Next() {
		var (
			s       model.PlayerScore
			weights []byte
		)
		if err := rows.Scan(&s.ID, &s.PlayerID, &s.StatisticsID, &s.StrategyID, &weights, &s.RawScore, &s.Percentile, &s.HistogramID, &s.ComputedAt); err != nil {
			return nil, fmt.Errorf("scan player score: %w", err)
		}
		if err := json.Unmarshal(weights, &s.Weights); err != nil {
			return nil, fmt.Errorf("unmarshal weights: %w", err)
		}
		scores = append(scores, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("player scores rows error: %w", err)
	}

	return scores, nil
}
