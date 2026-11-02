package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type ScoreHistogramSql struct {
	db *sql.DB
}

func NewScoreHistogramDao(db *sql.DB) *ScoreHistogramSql {
	return &ScoreHistogramSql{db: db}
}

func (d *ScoreHistogramSql) Save(ctx context.Context, h model.ScoreHistogram) (int64, error) {
	if d.db == nil {
		return 0, errNilDatabase
	}

	const query = `INSERT INTO scoring_histograms (position, league_code, buckets) VALUES ($1, $2, $3) RETURNING id`

	var id int64
	if err := d.db.QueryRowContext(ctx, query, h.Position, h.LeagueCode, pq.Array(h.Buckets)).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert score histogram: %w", err)
	}

	return id, nil
}

func (d *ScoreHistogramSql) GetByScope(ctx context.Context, scope model.ScoreScope) (model.ScoreHistogram, error) {
	if d.db == nil {
		return model.ScoreHistogram{}, errNilDatabase
	}

	const query = `SELECT id, position, league_code, buckets, computed_at
		FROM scoring_histograms WHERE position = $1 AND league_code = $2 ORDER BY computed_at DESC, id DESC LIMIT 1`

	var (
		h       model.ScoreHistogram
		buckets pq.Int64Array
	)
	if err := d.db.QueryRowContext(ctx, query, scope.Position, scope.LeagueCode).Scan(&h.ID, &h.Position, &h.LeagueCode, &buckets, &h.ComputedAt); err != nil {
		return model.ScoreHistogram{}, fmt.Errorf("get score histogram: %w", err)
	}
	h.Buckets = []int64(buckets)

	return h, nil
}
