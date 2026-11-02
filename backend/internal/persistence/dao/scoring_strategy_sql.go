package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type ScoringStrategySql struct {
	db *sql.DB
}

func NewScoringStrategyDao(db *sql.DB) *ScoringStrategySql {
	return &ScoringStrategySql{db: db}
}

func (d *ScoringStrategySql) List(ctx context.Context) ([]model.ScoringStrategy, error) {
	if d.db == nil {
		return nil, errNilDatabase
	}

	const query = `SELECT id, name, alpha, weights FROM scoring_strategies ORDER BY id`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query scoring strategies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	strategies := make([]model.ScoringStrategy, 0)
	for rows.Next() {
		var (
			s       model.ScoringStrategy
			weights []byte
		)
		if err := rows.Scan(&s.ID, &s.Name, &s.Alpha, &weights); err != nil {
			return nil, fmt.Errorf("scan scoring strategy: %w", err)
		}
		if err := json.Unmarshal(weights, &s.Weights); err != nil {
			return nil, fmt.Errorf("unmarshal weights: %w", err)
		}
		strategies = append(strategies, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scoring strategies rows error: %w", err)
	}

	return strategies, nil
}
