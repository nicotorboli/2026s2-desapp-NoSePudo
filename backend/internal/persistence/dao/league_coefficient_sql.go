package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type LeagueCoefficientSql struct {
	db *sql.DB
}

func NewLeagueCoefficientDao(db *sql.DB) *LeagueCoefficientSql {
	return &LeagueCoefficientSql{db: db}
}

func (d *LeagueCoefficientSql) Insert(ctx context.Context, coefficient model.LeagueCoefficient) error {
	if d.db == nil {
		return errNilDatabase
	}

	const query = `INSERT INTO league_coefficients (league_code, coefficient, set_by) VALUES ($1, $2, $3)`

	if _, err := d.db.ExecContext(ctx, query, coefficient.LeagueCode, coefficient.Coefficient, coefficient.SetBy); err != nil {
		return fmt.Errorf("insert league coefficient: %w", err)
	}

	return nil
}

func (d *LeagueCoefficientSql) CurrentFor(ctx context.Context, leagueCode string) (model.LeagueCoefficient, bool, error) {
	if d.db == nil {
		return model.LeagueCoefficient{}, false, errNilDatabase
	}

	const query = `SELECT id, league_code, coefficient, effective_from, set_by
		FROM league_coefficients WHERE league_code = $1 ORDER BY effective_from DESC, id DESC LIMIT 1`

	var c model.LeagueCoefficient
	if err := d.db.QueryRowContext(ctx, query, leagueCode).Scan(&c.ID, &c.LeagueCode, &c.Coefficient, &c.EffectiveFrom, &c.SetBy); err != nil {
		if err == sql.ErrNoRows {
			return model.LeagueCoefficient{}, false, nil
		}
		return model.LeagueCoefficient{}, false, fmt.Errorf("get current league coefficient: %w", err)
	}

	return c, true, nil
}

func (d *LeagueCoefficientSql) AtOrBefore(ctx context.Context, leagueCode string, at time.Time) (model.LeagueCoefficient, bool, error) {
	if d.db == nil {
		return model.LeagueCoefficient{}, false, errNilDatabase
	}

	const query = `SELECT id, league_code, coefficient, effective_from, set_by
		FROM league_coefficients WHERE league_code = $1 AND effective_from <= $2
		ORDER BY effective_from DESC, id DESC LIMIT 1`

	var c model.LeagueCoefficient
	if err := d.db.QueryRowContext(ctx, query, leagueCode, at).Scan(
		&c.ID,
		&c.LeagueCode,
		&c.Coefficient,
		&c.EffectiveFrom,
		&c.SetBy,
	); err != nil {
		if err == sql.ErrNoRows {
			return model.LeagueCoefficient{}, false, nil
		}
		return model.LeagueCoefficient{}, false, fmt.Errorf("get league coefficient at date: %w", err)
	}

	return c, true, nil
}
