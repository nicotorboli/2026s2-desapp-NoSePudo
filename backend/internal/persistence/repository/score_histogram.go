package repository

import (
	"context"
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type ScoreHistogramDAO interface {
	Save(ctx context.Context, h model.ScoreHistogram) (int64, error)
	GetByScope(ctx context.Context, scope model.ScoreScope) (model.ScoreHistogram, error)
}

type ScoreHistogramRepositoryImpl struct {
	dao ScoreHistogramDAO
}

func NewScoreHistogramRepository(dao ScoreHistogramDAO) *ScoreHistogramRepositoryImpl {
	return &ScoreHistogramRepositoryImpl{dao: dao}
}

func (r *ScoreHistogramRepositoryImpl) Save(ctx context.Context, h model.ScoreHistogram) (int64, error) {
	id, err := r.dao.Save(ctx, h)
	if err != nil {
		return 0, fmt.Errorf("repository save score histogram: %w", err)
	}
	return id, nil
}

func (r *ScoreHistogramRepositoryImpl) Get(ctx context.Context, scope model.ScoreScope) (model.ScoreHistogram, error) {
	return r.dao.GetByScope(ctx, scope)
}
