package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"

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
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare insert player score: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for i := range scores {
		weights, err := json.Marshal(scores[i].Weights)
		if err != nil {
			return fmt.Errorf("marshal weights: %w", err)
		}
		if err := stmt.QueryRowContext(
			ctx,
			scores[i].PlayerID,
			scores[i].StatisticsID,
			scores[i].StrategyID,
			weights,
			scores[i].RawScore,
			scores[i].Percentile,
			scores[i].HistogramID,
		).Scan(&scores[i].ID); err != nil {
			return fmt.Errorf("insert player score for player %d: %w", scores[i].PlayerID, err)
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

func (d *PlayerScoreSql) LatestPercentiles(ctx context.Context, playerIDs []int64, strategyID ...int) (map[int64]float64, error) {
	if d.db == nil {
		return nil, errNilDatabase
	}

	query := `
		SELECT DISTINCT ON (player_id) player_id, percentile
		FROM player_scores
		WHERE player_id = ANY($1)
		ORDER BY player_id, computed_at DESC, id DESC`
	args := []any{pq.Array(playerIDs)}

	if len(strategyID) > 0 && strategyID[0] > 0 {
		query = `
			SELECT DISTINCT ON (player_id) player_id, percentile
			FROM player_scores
			WHERE player_id = ANY($1) AND strategy_id = $2
			ORDER BY player_id, computed_at DESC, id DESC`
		args = append(args, strategyID[0])
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query latest percentiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	percentiles := make(map[int64]float64)
	for rows.Next() {
		var (
			playerID   int64
			percentile float64
		)
		if err := rows.Scan(&playerID, &percentile); err != nil {
			return nil, fmt.Errorf("scan latest percentile: %w", err)
		}
		percentiles[playerID] = percentile
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("latest percentiles rows error: %w", err)
	}

	return percentiles, nil
}
