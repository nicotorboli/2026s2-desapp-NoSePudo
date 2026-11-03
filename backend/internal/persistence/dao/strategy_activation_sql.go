package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type StrategyActivationSql struct {
	db *sql.DB
}

func NewStrategyActivationDao(db *sql.DB) *StrategyActivationSql {
	return &StrategyActivationSql{db: db}
}

func (d *StrategyActivationSql) Insert(ctx context.Context, activation model.StrategyActivation) error {
	if d.db == nil {
		return errNilDatabase
	}

	const query = `INSERT INTO scoring_strategy_activations (strategy_id, activated_by) VALUES ($1, $2)`

	if _, err := d.db.ExecContext(ctx, query, activation.StrategyID, activation.ActivatedBy); err != nil {
		return fmt.Errorf("insert strategy activation: %w", err)
	}

	return nil
}

func (d *StrategyActivationSql) Latest(ctx context.Context) (model.StrategyActivation, bool, error) {
	if d.db == nil {
		return model.StrategyActivation{}, false, errNilDatabase
	}

	const query = `SELECT id, strategy_id, activated_at, activated_by
		FROM scoring_strategy_activations ORDER BY activated_at DESC, id DESC LIMIT 1`

	var a model.StrategyActivation
	if err := d.db.QueryRowContext(ctx, query).Scan(&a.ID, &a.StrategyID, &a.ActivatedAt, &a.ActivatedBy); err != nil {
		if err == sql.ErrNoRows {
			return model.StrategyActivation{}, false, nil
		}
		return model.StrategyActivation{}, false, fmt.Errorf("get latest strategy activation: %w", err)
	}

	return a, true, nil
}
